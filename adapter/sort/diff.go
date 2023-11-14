package sort

//在a中,不在b中
func DiffStrSlice(a, b []string) []string {
	bm := map[string]bool{}
	for _, s := range b {
		bm[s] = true
	}
	r := []string{}
	for _, s := range a {
		if _, ok := bm[s]; !ok {
			r = append(r, s)
		}
	}
	return r
}
