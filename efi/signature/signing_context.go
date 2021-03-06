package signature

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/binary"
	"fmt"
	"log"

	"github.com/foxboron/goefi/efi/attributes"
	"github.com/foxboron/goefi/efi/util"
	"github.com/foxboron/pkcs7"
)

// Handles the values we use for EFI Variable signatures
type SigningContext struct {
	Cert    *x509.Certificate
	Key     *rsa.PrivateKey
	Varname []byte
	Attr    attributes.Attributes
	Guid    util.EFIGUID
	Data    []byte
}

// Uses EFIVariableAuthentication2
// Section 8.2.2 - Using the EFI_VARIABLE_AUTHENTICATION_2 descriptor
func NewSignedEFIVariable(ctx *SigningContext) *EFIVariableAuthentication2 {
	buf := new(bytes.Buffer)
	efva := NewEFIVariableAuthentication2()
	// The order is important
	// TODO: Expose the Time variable
	time := util.EFITime{Year: 2020,
		Month:      4,
		Day:        15,
		Hour:       21,
		Minute:     1,
		Second:     45,
		Pad1:       0,
		Nanosecond: 0,
		TimeZone:   0,
		Daylight:   0,
		Pad2:       0}
	writeOrder := []interface{}{
		ctx.Varname,
		ctx.Guid,
		ctx.Attr,
		time,
		ctx.Data,
	}
	fmt.Printf("%+v", writeOrder)
	// BIO_write(data_bio, ctx->var_name, ctx->var_name_bytes);
	// BIO_write(data_bio, &ctx->var_guid, sizeof(ctx->var_guid));
	// BIO_write(data_bio, &ctx->var_attrs, sizeof(ctx->var_attrs));
	// BIO_write(data_bio, &timestamp, sizeof(timestamp));
	// BIO_write(data_bio, ctx->data, ctx->data_len);
	for _, d := range writeOrder {
		if err := binary.Write(buf, binary.LittleEndian, d); err != nil {
			log.Fatal(err)
		}
	}
	sd, err := pkcs7.NewSignedData(buf.Bytes())
	if err != nil {
		log.Fatal(err)
	}

	// Page 246

	// SignedData.digestAlgorithms shall contain the digest algorithm used when
	// preparing the signature.  Ognly a digest algorithm of SHA-256 is accepted

	// SignerInfo.digestEncryptionAlgorithm shall be set to the algorithm used to
	// sign the data. Only a digest encryption algorithm of rSA with PKCS #1 v1.5
	// padding (RSASSA_PKCS1v1_5). is accepted.

	// Apparently we don't get the correct message diest if we set the
	// DigestAlgorithm to SHA256. However, we still get SHA256 one place and RSA
	// for message digest another if we do this. So maybe it works. I have no
	// clue.

	// sd.SetDigestAlgorithm(pkcs7.OIDEncryptionAlgorithmRSA)
	sd.SetDigestAlgorithm(pkcs7.OIDDigestAlgorithmSHA256)
	sd.SetEncryptionAlgorithm(pkcs7.OIDEncryptionAlgorithmRSA)

	sd.RemoveUnauthenticatedAttributes()
	if err := sd.AddSigner(ctx.Cert, ctx.Key, pkcs7.SignerInfoConfig{}); err != nil {
		log.Fatalf("Cannot add signer: %s", err)
	}
	sd.Detach()
	detachedSignature, err := sd.Finish()
	if err != nil {
		log.Fatal(err)
	}
	detachedSignature = util.PatchASN1(detachedSignature)
	efva.AuthInfo.Header.Length += uint32(len(detachedSignature))
	efva.AuthInfo.CertData = detachedSignature
	return efva
}
