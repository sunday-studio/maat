package main

import "github.com/sunday-studio/maat/internal/maatui"

func tuiCommand(args []string) error {
	store, err := loadStore(args)
	if err != nil {
		return err
	}
	cfg, err := readConfig()
	options := maatui.TUIOptions{}
	if err == nil {
		options.AutoPullBeforeRefresh = cfg.AutoPullBeforeRead
	}
	return maatui.RunTUIWithOptions(store, options)
}
