// Copyright 2015 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

// +build !darwin,!android

package libkb

func NewSecretStore(c SecretStoreContext, username NormalizedUsername) SecretStore {
	return nil
}

func HasSecretStore() bool {
	return false
}

func GetUsersWithStoredSecrets(c SecretStoreContext) ([]string, error) {
	return nil, nil
}

func GetTerminalPrompt() string {
	// TODO: Come up with specific prompts for other platforms.
	return "Store your key in the local secret store?"
}
