package main

import (
	"fmt"
	"slices"
)

// trash is the folder accounts deleted go to first, and are deleted from
// for good.
const trash = 2

// folder is a category of accounts, as fhtool's.
type folder struct {
	ID   int
	Name string
}

// store holds the accounts and their folders, as fhtool's database does.
// Only the main thread touches it.
type store struct {
	accounts []account
	folders  []folder
	index    map[int]int // where each account ID is in accounts
	nextID   int
	seed     uint64
}

// newStore makes n accounts in the first folder.
func newStore(n int, seed uint64) *store {
	s := &store{
		accounts: generate(n, 0, seed),
		folders:  []folder{{0, "Mặc định"}, {1, "Thư mục 2"}, {trash, "Thùng rác"}},
		nextID:   n,
		seed:     seed,
	}
	s.reindex()
	return s
}

// reindex finds the accounts again after some were removed.
func (s *store) reindex() {
	s.index = make(map[int]int, len(s.accounts))
	for i := range s.accounts {
		s.index[s.accounts[i].ID] = i
	}
}

// add makes one more account, in folder.
func (s *store) add(folder int) {
	a := generate(1, s.nextID, s.seed)[0]
	a.Folder = folder
	s.index[a.ID] = len(s.accounts)
	s.accounts = append(s.accounts, a)
	s.nextID++
}

// move puts the accounts ids in folder.
func (s *store) move(ids []int, folder int) {
	for _, id := range ids {
		if i, ok := s.index[id]; ok {
			s.accounts[i].Folder = folder
		}
	}
}

// remove deletes the accounts ids for good.
func (s *store) remove(ids []int) {
	gone := make(map[int]bool, len(ids))
	for _, id := range ids {
		gone[id] = true
	}
	s.accounts = slices.DeleteFunc(s.accounts, func(a account) bool { return gone[a.ID] })
	s.reindex()
}

// dedupe deletes those of the accounts ids whose UID one before them in
// ids has, and returns their IDs.
func (s *store) dedupe(ids []int) []int {
	seen := map[string]bool{}
	var dups []int
	for _, id := range ids {
		i, ok := s.index[id]
		if !ok {
			continue
		}
		if uid := s.accounts[i].UID; seen[uid] {
			dups = append(dups, id)
		} else {
			seen[uid] = true
		}
	}
	if len(dups) > 0 {
		s.remove(dups)
	}
	return dups
}

// folderByName is the folder named name.
func (s *store) folderByName(name string) (folder, bool) {
	for _, f := range s.folders {
		if f.Name == name {
			return f, true
		}
	}
	return folder{}, false
}

// names are the folders' names, for the folder select.
func (s *store) names() []string {
	names := make([]string, len(s.folders))
	for i, f := range s.folders {
		names[i] = f.Name
	}
	return names
}

// addFolder makes a folder, named for its ID.
func (s *store) addFolder() folder {
	id := 0
	for _, f := range s.folders {
		id = max(id, f.ID+1)
	}
	f := folder{id, fmt.Sprintf("Thư mục %d", id)}
	s.folders = append(s.folders, f)
	return f
}

// renameFolder gives folder id another name, and returns it: the example
// has no dialog to type one in.
func (s *store) renameFolder(id int) string {
	for i := range s.folders {
		if s.folders[i].ID == id {
			s.folders[i].Name += " (đổi tên)"
			return s.folders[i].Name
		}
	}
	return ""
}

// deleteFolder deletes folder id, its accounts going to the trash, and
// reports whether it did: the trash stays.
func (s *store) deleteFolder(id int) bool {
	if id == trash {
		return false
	}
	for i := range s.accounts {
		if s.accounts[i].Folder == id {
			s.accounts[i].Folder = trash
		}
	}
	s.folders = slices.DeleteFunc(s.folders, func(f folder) bool { return f.ID == id })
	return true
}
