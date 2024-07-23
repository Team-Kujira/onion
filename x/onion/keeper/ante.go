package keeper

import (
	"bytes"
	"fmt"

	"onion/x/onion/types"

	errorsmod "cosmossdk.io/errors"
	kmultisig "github.com/cosmos/cosmos-sdk/crypto/keys/multisig"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	// txsigning "github.com/cosmos/cosmos-sdk/types/tx/signing"
	txsigning "cosmossdk.io/x/tx/signing"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"google.golang.org/protobuf/types/known/anypb"
)

func GetSignerAcc(ctx sdk.Context, ak types.AccountKeeper, addr sdk.AccAddress) (authtypes.AccountI, error) {
	if acc := ak.GetAccount(ctx, addr); acc != nil {
		return acc, nil
	}

	return nil, errorsmod.Wrapf(sdkerrors.ErrUnknownAddress, "account %s does not exist", addr)
}

// CountSubKeys counts the total number of keys for a multi-sig public key.
func CountSubKeys(pub cryptotypes.PubKey) int {
	v, ok := pub.(*kmultisig.LegacyAminoPubKey)
	if !ok {
		return 1
	}

	numKeys := 0
	for _, subkey := range v.GetPubKeys() {
		numKeys += CountSubKeys(subkey)
	}

	return numKeys
}

func OnlyLegacyAminoSigners(sigData signing.SignatureData) bool {
	switch v := sigData.(type) {
	case *signing.SingleSignatureData:
		return v.SignMode == signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON
	case *signing.MultiSignatureData:
		for _, s := range v.Signatures {
			if !OnlyLegacyAminoSigners(s) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (k Keeper) ExecuteAnte(ctx sdk.Context, tx sdk.Tx) error {
	// ValidateBasicDecorator
	if validateBasic, ok := tx.(sdk.HasValidateBasic); ok {
		if err := validateBasic.ValidateBasic(); err != nil {
			return err
		}
	}

	// SetPubKeyDecorator
	sigTx, ok := tx.(authsigning.SigVerifiableTx)
	if !ok {
		return errorsmod.Wrap(sdkerrors.ErrTxDecode, "invalid tx type")
	}

	pubkeys, err := sigTx.GetPubKeys()
	if err != nil {
		return err
	}
	signers, err := sigTx.GetSigners()
	if err != nil {
		return err
	}

	for i, pk := range pubkeys {
		if pk == nil {
			continue
		}
		if !bytes.Equal(pk.Address(), signers[i]) {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidPubKey,
				"pubKey does not match signer address %s with signer index: %d", signers[i], i)
		}

		acc, err := GetSignerAcc(ctx, k.accountKeeper, signers[i])
		if err != nil {
			acc = k.accountKeeper.NewAccountWithAddress(ctx, signers[i])
			k.accountKeeper.SetAccount(ctx, acc)
		}
		if acc.GetPubKey() != nil {
			continue
		}
		err = acc.SetPubKey(pk)
		if err != nil {
			return errorsmod.Wrap(sdkerrors.ErrInvalidPubKey, err.Error())
		}
		k.accountKeeper.SetAccount(ctx, acc)
	}

	// ValidateSigCountDecorator
	params := k.accountKeeper.GetParams(ctx)
	pubKeys, err := sigTx.GetPubKeys()
	if err != nil {
		return err
	}

	sigCount := 0
	for _, pk := range pubKeys {
		sigCount += CountSubKeys(pk)
		if uint64(sigCount) > params.TxSigLimit {
			return errorsmod.Wrapf(sdkerrors.ErrTooManySignatures,
				"signatures: %d, limit: %d", sigCount, params.TxSigLimit)
		}
	}

	// SigVerificationDecorator
	sigs, err := sigTx.GetSignaturesV2()
	if err != nil {
		return err
	}

	signerAddrs, err := sigTx.GetSigners()
	if err != nil {
		return err
	}

	if len(sigs) != len(signerAddrs) {
		return errorsmod.Wrapf(sdkerrors.ErrUnauthorized, "invalid number of signer;  expected: %d, got %d", len(signerAddrs), len(sigs))
	}

	for i, sig := range sigs {
		acc, err := GetSignerAcc(ctx, k.accountKeeper, signerAddrs[i])
		if err != nil {
			return err
		}

		pubKey := acc.GetPubKey()
		if pubKey == nil {
			return errorsmod.Wrap(sdkerrors.ErrInvalidPubKey, "pubkey on account is not set")
		}

		onionSeq := uint64(0)
		seq, err := k.GetSequence(ctx, acc.GetAddress().String())
		if err == nil {
			onionSeq = seq.Sequence
		}

		if sig.Sequence != onionSeq {
			return errorsmod.Wrapf(
				sdkerrors.ErrWrongSequence,
				"onion sequence mismatch, expected %d, got %d", onionSeq, sig.Sequence,
			)
		}

		chainID := ctx.ChainID()
		accNum := types.AccountNumber
		anyPk, _ := codectypes.NewAnyWithValue(pubKey)
		signerData := txsigning.SignerData{
			Address:       acc.GetAddress().String(),
			ChainID:       chainID,
			AccountNumber: accNum,
			Sequence:      acc.GetSequence(),
			PubKey: &anypb.Any{
				TypeUrl: anyPk.TypeUrl,
				Value:   anyPk.Value,
			},
		}

		adaptableTx, ok := tx.(authsigning.V2AdaptableTx)
		if !ok {
			return fmt.Errorf("expected tx to implement V2AdaptableTx, got %T", tx)
		}
		txData := adaptableTx.GetSigningTxData()
		err = authsigning.VerifySignature(ctx, pubKey, signerData, sig.Data, k.SignModeHandler, txData)
		if err != nil {
			var errMsg string
			if OnlyLegacyAminoSigners(sig.Data) {
				errMsg = fmt.Sprintf("signature verification failed; please verify account number (%d), sequence (%d) and chain-id (%s)", accNum, acc.GetSequence(), chainID)
			} else {
				errMsg = fmt.Sprintf("signature verification failed; please verify account number (%d) and chain-id (%s)", accNum, chainID)
			}
			return errorsmod.Wrap(sdkerrors.ErrUnauthorized, errMsg)
		}
	}

	// IncrementSequenceDecorator
	for _, addr := range signerAddrs {
		seq, err := k.GetSequence(ctx, sdk.AccAddress(addr).String())
		if err != nil {
			seq = types.OnionSequence{
				Address:  sdk.AccAddress(addr).String(),
				Sequence: 0,
			}
		}

		seq.Sequence++
		err = k.SetSequence(ctx, seq)
		if err != nil {
			return err
		}
	}

	return nil
}
