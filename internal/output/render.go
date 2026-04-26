package output

// Render is the unified entry point: in human mode it renders via
// `humanFn`, in JSON mode it serialises `payload`. When `humanFn` is
// nil and the format is human, it falls back to pretty JSON — legible
// to humans and parseable by machines, while a nil-fn fallback that
// emitted a Go struct literal would be neither.
func (r *Renderer) Render(payload any, humanFn func() error) error {
	if r.Format == FormatJSON {
		return r.JSON(payload)
	}
	if humanFn != nil {
		return humanFn()
	}
	if r.Quiet {
		return nil
	}
	return r.JSON(payload)
}
