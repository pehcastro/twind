package render

func (t *Tree) Cascades() int { return t.cascades }

func (t *Tree) Visits() int { return t.visits }

func (t *Tree) Reaches(path []int) bool {
	s, _ := t.find(path)
	return s != nil
}
