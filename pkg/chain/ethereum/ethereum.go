// Package ethereum contains implementation of ethereum chain interface.
package ethereum

import (
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ipfs/go-log"

	"github.com/keep-network/keep-common/pkg/chain/ethereum/ethutil"
	"github.com/keep-network/keep-common/pkg/subscription"
	"github.com/keep-network/keep-core/pkg/chain"
	ecdsachain "github.com/keep-network/keep-ecdsa/pkg/chain"
	"github.com/keep-network/keep-ecdsa/pkg/chain/gen/contract"
	"github.com/keep-network/keep-ecdsa/pkg/ecdsa"
	"github.com/keep-network/keep-ecdsa/pkg/utils/byteutils"
)

var logger = log.Logger("keep-chain-eth-ethereum")

// Address returns client's ethereum address.
func (ec *EthereumChain) Address() common.Address {
	return ec.accountKey.Address
}

// RegisterAsMemberCandidate registers client as a candidate to be selected
// to a keep.
func (ec *EthereumChain) RegisterAsMemberCandidate(application common.Address) error {
	transaction, err := ec.bondedECDSAKeepFactoryContract.RegisterMemberCandidate(
		application,
	)
	if err != nil {
		return err
	}

	logger.Debugf("submitted RegisterMemberCandidate transaction with hash: [%x]", transaction.Hash())

	return nil
}

// OnBondedECDSAKeepCreated installs a callback that is invoked when an on-chain
// notification of a new ECDSA keep creation is seen.
func (ec *EthereumChain) OnBondedECDSAKeepCreated(
	handler func(event *ecdsachain.BondedECDSAKeepCreatedEvent),
) (subscription.EventSubscription, error) {
	return ec.bondedECDSAKeepFactoryContract.WatchBondedECDSAKeepCreated(
		func(
			KeepAddress common.Address,
			Members []common.Address,
			Owner common.Address,
			Application common.Address,
			blockNumber uint64,
		) {
			handler(&ecdsachain.BondedECDSAKeepCreatedEvent{
				KeepAddress: KeepAddress,
				Members:     Members,
			})
		},
		func(err error) error {
			return fmt.Errorf("watch keep created failed: [%v]", err)
		},
	)
}

// OnKeepClosed installs a callback that is invoked on-chain when keep is closed.
func (ec *EthereumChain) OnKeepClosed(
	keepAddress common.Address,
	handler func(event *ecdsachain.KeepClosedEvent),
) (subscription.EventSubscription, error) {
	keepContract, err := ec.getKeepContract(keepAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to create contract abi: [%v]", err)
	}
	return keepContract.WatchKeepClosed(
		func(blockNumber uint64) {
			handler(&ecdsachain.KeepClosedEvent{BlockNumber: blockNumber})
		},
		func(err error) error {
			return fmt.Errorf("keep closed callback failed: [%v]", err)
		},
	)
}

// OnKeepTerminated installs a callback that is invoked on-chain when keep
// is terminated.
func (ec *EthereumChain) OnKeepTerminated(
	keepAddress common.Address,
	handler func(event *ecdsachain.KeepTerminatedEvent),
) (subscription.EventSubscription, error) {
	keepContract, err := ec.getKeepContract(keepAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to create contract abi: [%v]", err)
	}
	return keepContract.WatchKeepTerminated(
		func(blockNumber uint64) {
			handler(&ecdsachain.KeepTerminatedEvent{BlockNumber: blockNumber})
		},
		func(err error) error {
			return fmt.Errorf("keep terminated callback failed: [%v]", err)
		},
	)
}

// OnPublicKeyPublished installs a callback that is invoked when an on-chain
// event of a published public key was emitted.
func (ec *EthereumChain) OnPublicKeyPublished(
	keepAddress common.Address,
	handler func(event *ecdsachain.PublicKeyPublishedEvent),
) (subscription.EventSubscription, error) {
	keepContract, err := ec.getKeepContract(keepAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to create contract abi: [%v]", err)
	}

	return keepContract.WatchPublicKeyPublished(
		func(
			PublicKey []byte,
			blockNumber uint64,
		) {
			handler(&ecdsachain.PublicKeyPublishedEvent{
				PublicKey: PublicKey,
			})
		},
		func(err error) error {
			return fmt.Errorf("keep created callback failed: [%v]", err)
		},
	)
}

// OnConflictingPublicKeySubmitted installs a callback that is invoked when an
// on-chain notification of a conflicting public key submission is seen.
func (ec *EthereumChain) OnConflictingPublicKeySubmitted(
	keepAddress common.Address,
	handler func(event *ecdsachain.ConflictingPublicKeySubmittedEvent),
) (subscription.EventSubscription, error) {
	keepContract, err := ec.getKeepContract(keepAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to create contract abi: [%v]", err)
	}

	return keepContract.WatchConflictingPublicKeySubmitted(
		func(
			SubmittingMember common.Address,
			ConflictingPublicKey []byte,
			blockNumber uint64,
		) {
			handler(&ecdsachain.ConflictingPublicKeySubmittedEvent{
				SubmittingMember:     SubmittingMember,
				ConflictingPublicKey: ConflictingPublicKey,
			})
		},
		func(err error) error {
			return fmt.Errorf("keep created callback failed: [%v]", err)
		},
	)
}

// OnSignatureRequested installs a callback that is invoked on-chain
// when a keep's signature is requested.
func (ec *EthereumChain) OnSignatureRequested(
	keepAddress common.Address,
	handler func(event *ecdsachain.SignatureRequestedEvent),
) (subscription.EventSubscription, error) {
	keepContract, err := ec.getKeepContract(keepAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to create contract abi: [%v]", err)
	}

	return keepContract.WatchSignatureRequested(
		func(
			Digest [32]uint8,
			blockNumber uint64,
		) {
			handler(&ecdsachain.SignatureRequestedEvent{
				Digest:      Digest,
				BlockNumber: blockNumber,
			})
		},
		func(err error) error {
			return fmt.Errorf("keep signature requested callback failed: [%v]", err)
		},
	)
}

// SubmitKeepPublicKey submits a public key to a keep contract deployed under
// a given address.
func (ec *EthereumChain) SubmitKeepPublicKey(
	keepAddress common.Address,
	publicKey [64]byte,
) error {
	keepContract, err := ec.getKeepContract(keepAddress)
	if err != nil {
		return err
	}

	submitPubKey := func() error {
		transaction, err := keepContract.SubmitPublicKey(
			publicKey[:],
			ethutil.TransactionOptions{
				GasLimit: 3000000, // enough for a group size of 16
			},
		)
		if err != nil {
			return err
		}

		logger.Debugf("submitted SubmitPublicKey transaction with hash: [%x]", transaction.Hash())
		return nil
	}

	// There might be a scenario, when a public key submission fails because of
	// a new cloned contract has not been registered by the ethereum node. Common
	// case is when Ethereum nodes are behind a load balancer and not fully synced
	// with each other. To mitigate this issue, a client will retry submitting
	// a public key up to 4 times with a 250ms interval.
	if err := ec.withRetry(submitPubKey); err != nil {
		return err
	}

	return nil
}

func (ec *EthereumChain) withRetry(fn func() error) error {
	const numberOfRetries = 10
	const delay = 12 * time.Second

	for i := 1; ; i++ {
		err := fn()
		if err != nil {
			logger.Errorf("Error occurred [%v]; on [%v] retry", err, i)
			if i == numberOfRetries {
				return err
			}
			time.Sleep(delay)
		} else {
			return nil
		}
	}
}

func (ec *EthereumChain) getKeepContract(address common.Address) (*contract.BondedECDSAKeep, error) {
	bondedECDSAKeepContract, err := contract.NewBondedECDSAKeep(
		address,
		ec.accountKey,
		ec.client,
		ec.transactionMutex,
	)
	if err != nil {
		return nil, err
	}

	return bondedECDSAKeepContract, nil
}

// SubmitSignature submits a signature to a keep contract deployed under a
// given address.
func (ec *EthereumChain) SubmitSignature(
	keepAddress common.Address,
	signature *ecdsa.Signature,
) error {
	keepContract, err := ec.getKeepContract(keepAddress)
	if err != nil {
		return err
	}

	signatureR, err := byteutils.BytesTo32Byte(signature.R.Bytes())
	if err != nil {
		return err
	}

	signatureS, err := byteutils.BytesTo32Byte(signature.S.Bytes())
	if err != nil {
		return err
	}

	transaction, err := keepContract.SubmitSignature(
		signatureR,
		signatureS,
		uint8(signature.RecoveryID),
	)
	if err != nil {
		return err
	}

	logger.Debugf("submitted SubmitSignature transaction with hash: [%x]", transaction.Hash())

	return nil
}

// IsAwaitingSignature checks if the keep is waiting for a signature to be
// calculated for the given digest.
func (ec *EthereumChain) IsAwaitingSignature(keepAddress common.Address, digest [32]byte) (bool, error) {
	keepContract, err := ec.getKeepContract(keepAddress)
	if err != nil {
		return false, err
	}

	return keepContract.IsAwaitingSignature(digest)
}

// IsActive checks for current state of a keep on-chain.
func (ec *EthereumChain) IsActive(keepAddress common.Address) (bool, error) {
	keepContract, err := ec.getKeepContract(keepAddress)
	if err != nil {
		return false, err
	}

	return keepContract.IsActive()
}

// HasMinimumStake returns true if the specified address is staked.  False will
// be returned if not staked.  If err != nil then it was not possible to determine
// if the address is staked or not.
func (ec *EthereumChain) HasMinimumStake(address common.Address) (bool, error) {
	return ec.bondedECDSAKeepFactoryContract.HasMinimumStake(address)
}

// BalanceOf returns the stake balance of the specified address.
func (ec *EthereumChain) BalanceOf(address common.Address) (*big.Int, error) {
	return ec.bondedECDSAKeepFactoryContract.BalanceOf(address)
}

func (ec *EthereumChain) BlockCounter() chain.BlockCounter {
	return ec.blockCounter
}

func (ec *EthereumChain) IsRegisteredForApplication(application common.Address) (bool, error) {
	return ec.bondedECDSAKeepFactoryContract.IsOperatorRegistered(
		ec.Address(),
		application,
	)
}

func (ec *EthereumChain) IsEligibleForApplication(application common.Address) (bool, error) {
	return ec.bondedECDSAKeepFactoryContract.IsOperatorEligible(
		ec.Address(),
		application,
	)
}

func (ec *EthereumChain) IsStatusUpToDateForApplication(application common.Address) (bool, error) {
	return ec.bondedECDSAKeepFactoryContract.IsOperatorUpToDate(
		ec.Address(),
		application,
	)
}

func (ec *EthereumChain) UpdateStatusForApplication(application common.Address) error {
	transaction, err := ec.bondedECDSAKeepFactoryContract.UpdateOperatorStatus(
		ec.Address(),
		application,
	)
	if err != nil {
		return err
	}

	logger.Debugf(
		"submitted UpdateOperatorStatus transaction with hash: [%x]",
		transaction.Hash(),
	)

	return nil
}

func (ec *EthereumChain) GetKeepCount() (*big.Int, error) {
	return ec.bondedECDSAKeepFactoryContract.GetKeepCount()
}

func (ec *EthereumChain) GetKeepAtIndex(
	keepIndex *big.Int,
) (common.Address, error) {
	return ec.bondedECDSAKeepFactoryContract.GetKeepAtIndex(keepIndex)
}

func (ec *EthereumChain) LatestDigest(keepAddress common.Address) ([32]byte, error) {
	keepContract, err := ec.getKeepContract(keepAddress)
	if err != nil {
		return [32]byte{}, err
	}

	return keepContract.Digest()
}

func (ec *EthereumChain) GetPublicKey(keepAddress common.Address) ([]uint8, error) {
	keepContract, err := ec.getKeepContract(keepAddress)
	if err != nil {
		return []uint8{}, err
	}

	return keepContract.GetPublicKey()
}

func (ec *EthereumChain) GetMembers(
	keepAddress common.Address,
) ([]common.Address, error) {
	keepContract, err := ec.getKeepContract(keepAddress)
	if err != nil {
		return []common.Address{}, err
	}

	return keepContract.GetMembers()
}

func (ec *EthereumChain) HasKeyGenerationTimedOut(
	keepAddress common.Address,
) (bool, error) {
	keepContract, err := ec.getKeepContract(keepAddress)
	if err != nil {
		return false, err
	}

	return keepContract.HasKeyGenerationTimedOut()
}