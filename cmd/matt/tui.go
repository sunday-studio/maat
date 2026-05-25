package main

import "github.com/sunday-studio/maat/internal/maatui"

func tuiCommand(args []string) error {
	store, err := loadStore(args)
	if err != nil {
		return err
	}
	return maatui.RunTUI(store)
}
