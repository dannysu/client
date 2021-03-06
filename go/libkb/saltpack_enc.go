// Copyright 2015 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

package libkb

import (
	"io"

	"github.com/keybase/client/go/saltpack"
)

type SaltpackEncryptArg struct {
	Source    io.Reader
	Sink      io.WriteCloser
	Receivers []NaclDHKeyPublic
	Sender    NaclDHKeyPair
	Binary    bool
}

// SaltpackEncrypt reads from the given source, encrypts it for the given
// receivers from the given sender, and writes it to sink.  If
// Binary is false, the data written to sink will be armored.
func SaltpackEncrypt(g *GlobalContext, arg *SaltpackEncryptArg) error {
	var receiverBoxKeys []saltpack.BoxPublicKey
	for _, k := range arg.Receivers {
		receiverBoxKeys = append(receiverBoxKeys, naclBoxPublicKey(k))
	}

	var bsk saltpack.BoxSecretKey
	if !arg.Sender.IsNil() {
		bsk = naclBoxSecretKey(arg.Sender)
	}

	var plainsink io.WriteCloser
	var err error
	if arg.Binary {
		plainsink, err = saltpack.NewEncryptStream(arg.Sink, bsk, receiverBoxKeys)
	} else {
		plainsink, err = saltpack.NewEncryptArmor62Stream(arg.Sink, bsk, receiverBoxKeys, KeybaseSaltpackBrand)
	}
	if err != nil {
		return err
	}

	n, err := io.Copy(plainsink, arg.Source)
	if err != nil {
		return err
	}

	g.Log.Debug("Encrypt: wrote %d bytes", n)

	if err := plainsink.Close(); err != nil {
		return err
	}
	if err := arg.Sink.Close(); err != nil {
		return err
	}

	return nil
}
