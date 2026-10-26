package scene

// FoldingForTest sets whether FoldTranslations was called and returns what
// restores it, for tests that compare scenes drawn both ways.
func FoldingForTest(on bool) (restore func()) {
	was := fold.Load()
	if on {
		f := foldFunc((*TranslationFolder).fold)
		fold.Store(&f)
	} else {
		fold.Store(nil)
	}
	return func() { fold.Store(was) }
}
