package core

func init() {
	RegisterExporter("null-export", func(c *ServiceConfig, s *Service) (Exporter, error) { return &NullExporter{}, nil })
}

// a Dummy exporter if we don't want one
type NullExporter struct{}

func (n *NullExporter) Flatten(dir string) error {
	return nil
}
func (n *NullExporter) Export(dir string, items []*Item) error {
	return nil
}
func (n *NullExporter) ExportItem(item *Item) error {
	return nil
}

var _ Exporter = (*NullExporter)(nil)
