package cli

func (z *ZVM) Use(ver string) error {
	return z.loadVersionCache()
}