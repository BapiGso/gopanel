package frpc

//var rootCmd = &cobra.Command{
//	Use:   "frpc",
//	Short: "frpc is the client of frp (https://github.com/fatedier/frp)",
//	RunE: func(cmd *cobra.Command, args []string) error {
//		if showVersion {
//			fmt.Println(version.Full())
//			return nil
//		}
//
//		// If cfgDir is not empty, run multiple frpc service for each config file in cfgDir.
//		// Note that it's only designed for testing. It's not guaranteed to be stable.
//		if cfgDir != "" {
//			_ = runMultipleClients(cfgDir)
//			return nil
//		}
//
//		// Do not show command usage here.
//		err := sub.runClient(cfgFile)
//		if err != nil {
//			fmt.Println(err)
//			os.Exit(1)
//		}
//		return nil
//	},
//}
