package keeper_test

import (
	"context"
	"testing"

	"onion/x/onion/types"

	txsigning "cosmossdk.io/x/tx/signing"
	"github.com/cosmos/cosmos-sdk/client"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"
)

func (s *KeeperTestSuite) TestExecuteTxMsgs() {
	privKey1 := secp256k1.GenPrivKeyFromSecret([]byte("test1"))
	pubKey1 := privKey1.PubKey()
	privKey2 := secp256k1.GenPrivKeyFromSecret([]byte("test2"))
	pubKey2 := privKey2.PubKey()

	addr1 := sdk.AccAddress(pubKey1.Address())
	addr2 := sdk.AccAddress(pubKey2.Address())

	msgSend1 := &banktypes.MsgSend{
		FromAddress: addr1.String(),
		ToAddress:   addr2.String(),
		Amount:      sdk.Coins{sdk.NewInt64Coin("test", 100)},
	}
	msgSend2 := &banktypes.MsgSend{
		FromAddress: addr1.String(),
		ToAddress:   addr2.String(),
		Amount:      sdk.Coins{sdk.NewInt64Coin("test", 200)},
	}
	msgSend3 := &banktypes.MsgSend{
		FromAddress: addr1.String(),
		ToAddress:   addr2.String(),
		Amount:      sdk.Coins{sdk.NewInt64Coin("test", 1000)},
	}

	specs := map[string]struct {
		msgs               []sdk.Msg
		expErr             bool
		expSenderBalance   sdk.Coins
		expReceiverBalance sdk.Coins
	}{
		"empty messages execution": {
			msgs:               []sdk.Msg{},
			expErr:             false,
			expSenderBalance:   sdk.Coins{sdk.NewInt64Coin("test", 500)},
			expReceiverBalance: sdk.Coins{},
		},
		"successful execution of a single message": {
			msgs:               []sdk.Msg{msgSend1},
			expErr:             false,
			expSenderBalance:   sdk.Coins{sdk.NewInt64Coin("test", 400)},
			expReceiverBalance: sdk.Coins{sdk.NewInt64Coin("test", 100)},
		},
		"successful execution of multiple messages": {
			msgs:               []sdk.Msg{msgSend1, msgSend2},
			expErr:             false,
			expSenderBalance:   sdk.Coins{sdk.NewInt64Coin("test", 200)},
			expReceiverBalance: sdk.Coins{sdk.NewInt64Coin("test", 300)},
		},
		"one execution failure in multiple messages": {
			msgs:               []sdk.Msg{msgSend1, msgSend3},
			expErr:             true,
			expSenderBalance:   sdk.Coins{},
			expReceiverBalance: sdk.Coins{},
		},
	}
	for msg, spec := range specs {
		spec := spec
		s.Run(msg, func() {
			s.SetupTest()
			coins := sdk.Coins{sdk.NewInt64Coin("test", 500)}
			err := s.App.BankKeeper.MintCoins(s.Ctx, minttypes.ModuleName, coins)
			s.Require().NoError(err)
			err = s.App.BankKeeper.SendCoinsFromModuleToAccount(s.Ctx, minttypes.ModuleName, addr1, coins)
			s.Require().NoError(err)

			tx := newTx(s.T(), s.App.TxConfig(), addr1, s.Ctx.ChainID(), types.AccountNumber, spec.msgs, 0, privKey1)
			results, err := s.App.OnionKeeper.ExecuteTxMsgs(s.Ctx, tx)
			if spec.expErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
				s.Require().Len(results, len(spec.msgs))
				senderBalance := s.App.BankKeeper.GetAllBalances(s.Ctx, addr1)
				s.Require().Equal(senderBalance.String(), spec.expSenderBalance.String())
				receiverBalance := s.App.BankKeeper.GetAllBalances(s.Ctx, addr2)
				s.Require().Equal(receiverBalance.String(), spec.expReceiverBalance.String())
			}
		})
	}
}

func newTx(t *testing.T, cfg client.TxConfig, addr sdk.AccAddress, chainId string, accountNumber uint64, msgs []sdk.Msg, nonce uint64, privKey *secp256k1.PrivKey) signing.Tx {
	builder := cfg.NewTxBuilder()
	builder.SetMsgs(msgs...)
	if len(msgs) > 0 {
		pubKey := privKey.PubKey()
		signModeHandler := cfg.SignModeHandler()
		defaultSignMode, err := signing.APISignModeToInternal(cfg.SignModeHandler().DefaultMode())
		require.NoError(t, err)

		err = builder.SetSignatures(
			signingtypes.SignatureV2{
				PubKey:   pubKey,
				Sequence: nonce,
				Data: &signingtypes.SingleSignatureData{
					SignMode:  defaultSignMode,
					Signature: []byte{},
				},
			},
		)
		require.NoError(t, err)

		anyPk, _ := codectypes.NewAnyWithValue(pubKey)
		signerData := txsigning.SignerData{
			Address:       addr.String(),
			ChainID:       chainId,
			AccountNumber: accountNumber,
			Sequence:      nonce,
			PubKey: &anypb.Any{
				TypeUrl: anyPk.TypeUrl,
				Value:   anyPk.Value,
			},
		}

		tx := builder.GetTx()

		adaptableTx, ok := tx.(authsigning.V2AdaptableTx)
		require.True(t, ok)
		txData := adaptableTx.GetSigningTxData()

		signBytes, err := signModeHandler.GetSignBytes(
			context.Background(),
			signModeHandler.DefaultMode(),
			signerData,
			txData,
		)
		require.NoError(t, err)
		sigBz, err := privKey.Sign(signBytes)
		require.NoError(t, err)

		err = builder.SetSignatures(
			signingtypes.SignatureV2{
				PubKey:   pubKey,
				Sequence: nonce,
				Data: &signingtypes.SingleSignatureData{
					SignMode:  defaultSignMode,
					Signature: sigBz,
				},
			},
		)
		require.NoError(t, err)
	}

	return builder.GetTx()
}
