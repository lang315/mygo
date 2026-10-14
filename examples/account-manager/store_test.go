package main

import (
	"slices"
	"testing"
)

// checkIndex fails t unless s.index finds each account where it is.
func checkIndex(t *testing.T, s *store) {
	t.Helper()
	if len(s.index) != len(s.accounts) {
		t.Fatalf("index of %d for %d accounts", len(s.index), len(s.accounts))
	}
	for i, a := range s.accounts {
		if s.index[a.ID] != i {
			t.Fatalf("account %d at %d, indexed at %d", a.ID, i, s.index[a.ID])
		}
	}
}

func TestStoreMovesAndRemoves(t *testing.T) {
	s := newStore(10, 1)
	checkIndex(t, s)
	ids := []int{s.accounts[2].ID, s.accounts[5].ID}
	s.move(ids, 1)
	if s.accounts[2].Folder != 1 || s.accounts[5].Folder != 1 || s.accounts[3].Folder != 0 {
		t.Fatal("move")
	}
	s.remove(ids)
	if len(s.accounts) != 8 {
		t.Fatalf("%d accounts left", len(s.accounts))
	}
	checkIndex(t, s)
	if _, ok := s.index[ids[0]]; ok {
		t.Fatal("a removed account still indexed")
	}
	s.add(1)
	if a := s.accounts[len(s.accounts)-1]; a.ID != 10 || a.Folder != 1 {
		t.Fatalf("added %+v", a)
	}
	checkIndex(t, s)
}

func TestStoreDedupe(t *testing.T) {
	s := newStore(5, 1)
	s.accounts[3].UID = s.accounts[1].UID
	ids := []int{s.accounts[0].ID, s.accounts[1].ID, s.accounts[3].ID}
	dup := s.accounts[3].ID
	if gone := s.dedupe(ids); !slices.Equal(gone, []int{dup}) {
		t.Fatalf("dedupe removed %v, want [%d]", gone, dup)
	}
	if len(s.accounts) != 4 {
		t.Fatalf("%d accounts left", len(s.accounts))
	}
	checkIndex(t, s)
}

func TestStoreFolders(t *testing.T) {
	s := newStore(3, 1)
	if !slices.Equal(s.names(), []string{"Mặc định", "Thư mục 2", "Thùng rác"}) {
		t.Fatalf("folders %q", s.names())
	}
	f := s.addFolder()
	if f.ID != 3 || f.Name != "Thư mục 3" {
		t.Fatalf("new folder %+v", f)
	}
	if name := s.renameFolder(f.ID); name != "Thư mục 3 (đổi tên)" {
		t.Fatalf("renamed %q", name)
	}
	s.move([]int{s.accounts[0].ID}, f.ID)
	if !s.deleteFolder(f.ID) {
		t.Fatal("folder not deleted")
	}
	if s.accounts[0].Folder != trash {
		t.Fatal("the accounts of a folder deleted are not in the trash")
	}
	if _, ok := s.folderByName("Thư mục 3 (đổi tên)"); ok {
		t.Fatal("folder still listed")
	}
	if s.deleteFolder(trash) {
		t.Fatal("the trash was deleted")
	}
}

func TestDeleteFolderFallsBack(t *testing.T) {
	s := newStore(1, 1)
	s.deleteFolder(0)
	s.deleteFolder(1)
	if !slices.Equal(s.names(), []string{"Thùng rác"}) || s.folders[0].ID != trash {
		t.Fatalf("folders %q", s.names())
	}
}
