package deployer

func withTestOpenObserve(opts Options) Options {
	if opts.OpenObserveRootEmail == "" {
		opts.OpenObserveRootEmail = "ops@example.com"
	}
	if opts.OpenObserveRootPassword == "" {
		opts.OpenObserveRootPassword = "Complexpass#123"
	}
	return opts
}
