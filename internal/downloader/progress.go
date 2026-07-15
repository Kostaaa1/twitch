package downloader

type Progress struct {
	ID    string
	Label string
	Bytes int64
	Error error
	Done  bool
	Total float64
}

func (dl *Downloader) notifyProgress(u *Unit, n int64) {
	if dl.notifyFn == nil {
		return
	}

	dl.notifyFn(Progress{
		ID:    u.GetID(),
		Label: u.GetLabel(),
		Error: u.GetError(),
		Total: u.total,
		Bytes: n,
		Done:  false,
	})
}

func (dl *Downloader) notifyDone(u *Unit) {
	if dl.notifyFn == nil {
		return
	}

	dl.notifyFn(Progress{
		ID:    u.GetID(),
		Label: u.GetLabel(),
		Error: u.GetError(),
		Total: u.total,
		Bytes: 0,
		Done:  true,
	})
}
