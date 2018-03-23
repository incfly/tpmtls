// Package tpmkey implements crypto.PrivateKey using a TPM.
package tpmkey

import (
	"crypto"
	"encoding/asn1"
	"io"
	"math/big"
	"sync"

	"github.com/google/go-tpm/tpm2"
	"github.com/google/go-tpm/tpmutil"
)

type privateKey struct {
	pub crypto.PublicKey

	mu   *sync.Mutex
	rwc  io.ReadWriteCloser
	h    tpmutil.Handle
	pass string
}

// Private represents a TPM-connected private key. This key can be used as
// PrivateKey in tls.Certificate.
type Private interface {
	crypto.Signer
	io.Closer
}

// TODO(awly): more universal constructor.

// Primary creates an ECDSA primary key under specified hierarchy.
//
// User must call Close on the returned key when done to free resources in the
// TPM. Exiting the process without calling Close doesn't free the resources.
func PrimaryECC(path string, hierarchy tpmutil.Handle) (Private, error) {
	rwc, err := tpmutil.OpenTPM(path)
	if err != nil {
		return nil, err
	}
	public := tpm2.Public{
		Type:       tpm2.AlgECC,
		NameAlg:    tpm2.AlgSHA1,
		Attributes: tpm2.FlagSign | tpm2.FlagSensitiveDataOrigin | tpm2.FlagUserWithAuth,
		ECCParameters: &tpm2.ECCParams{
			Sign: &tpm2.SigScheme{
				Alg:  tpm2.AlgECDSA,
				Hash: tpm2.AlgSHA256,
			},
			CurveID: tpm2.CurveNISTP256,
			Point:   tpm2.ECPoint{X: big.NewInt(0), Y: big.NewInt(0)},
		},
	}

	h, pub, err := tpm2.CreatePrimary(rwc, hierarchy, tpm2.PCRSelection{}, "", "", public)
	if err != nil {
		rwc.Close()
		return nil, err
	}

	return &privateKey{rwc: rwc, h: h, pub: pub, mu: &sync.Mutex{}}, nil
}

func (pk *privateKey) Close() error {
	pk.mu.Lock()
	defer pk.mu.Unlock()

	if pk.rwc == nil {
		return nil
	}
	if pk.h != 0 {
		tpm2.FlushContext(pk.rwc, pk.h)
	}

	err := pk.rwc.Close()
	pk.rwc = nil
	return err
}

type ecdsaSignature struct {
	R, S *big.Int
}

func (pk *privateKey) Sign(_ io.Reader, digest []byte, _ crypto.SignerOpts) ([]byte, error) {
	pk.mu.Lock()
	defer pk.mu.Unlock()

	sig, err := tpm2.Sign(pk.rwc, pk.h, pk.pass, digest)
	if err != nil {
		return nil, err
	}
	if sig.RSA != nil {
		return sig.RSA.Signature, nil
	}
	return asn1.Marshal(ecdsaSignature{sig.ECC.R, sig.ECC.S})
}

func (pk *privateKey) Public() crypto.PublicKey { return pk.pub }
