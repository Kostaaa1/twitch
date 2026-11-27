package downloader

type Progress struct {
	ID    string
	Label string
	Bytes int64
	Error error
	Done  bool
	Total float64
}

func (u *Unit) notifyProgress(n int64) {
	if u.notifyFn == nil {
		return
	}
	u.notifyFn(Progress{
		ID:    u.GetID(),
		Label: u.GetLabel(),
		Error: u.GetError(),
		Total: u.total,
		Bytes: n,
		Done:  false,
	})
}

func (u *Unit) notifyDone() {
	if u.notifyFn == nil {
		return
	}
	u.notifyFn(Progress{
		ID:    u.GetID(),
		Label: u.GetLabel(),
		Error: u.GetError(),
		Total: u.total,
		Bytes: 0,
		Done:  true,
	})
}
