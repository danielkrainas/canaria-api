package token

type actionSet struct {
	stringSet
}

func newActionSet(actions ...string) actionSet {
	return actionSet{newStringSet(actions...)}
}

func (s actionSet) contains(action string) bool {
	return s.stringSet.contains("*") || s.stringSet.contains(action)
}

func contains(ss []string, q string) bool {
	for _, s := range ss {
		if s == q {
			return true
		}
	}

	return false
}

type stringSet map[string]struct{}

func newStringSet(keys ...string) stringSet {
	ss := make(stringSet, len(keys))
	ss.add(keys...)
	return ss
}

func (ss stringSet) add(keys ...string) {
	for _, key := range keys {
		ss[key] = struct{}{}
	}
}

func (ss stringSet) contains(key string) bool {
	_, ok := ss[key]
	return ok
}

func (ss stringSet) keys() []string {
	keys := make([]string, 0, len(ss))
	for key := range ss {
		keys = append(keys, key)
	}

	return keys
}
