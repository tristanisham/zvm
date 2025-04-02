package cli

// Report takes a string and adds it to the instance's Report for logging.
func (z *ZVM) Report(msg string) {

}

func (z *ZVM) StartReport(cmd string, args []string, flags []string) {
	z.report.Command= cmd
	z.report.Args = args
	z.report.Flags = flags
}
