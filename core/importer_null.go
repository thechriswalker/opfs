package core

func init() {
	RegisterImporter("null-import", func(c *ServiceConfig, s *Service) (Importer, error) { return &NullImporter{}, nil })
}

type NullImporter struct{}

func (n *NullImporter) Import(opts *ImportOptions) chan error {
	out := make(chan error)
	close(out) //i think this can be done synchonously
	return out
}

func (n *NullImporter) Watch(opts *ImportOptions, shutdown chan struct{}) chan error {
	out := make(chan error)
	go func() {
		<-shutdown
		close(out)
	}()
	return out
}

var _ Importer = (*NullImporter)(nil)
