package lmdbscan

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/bmatsuo/lmdb-go/lmdb"
)

type errcheck func(err error) (ok bool)

var pIsNil = func(err error) bool { return err == nil }

func TestScanner_Scan(t *testing.T) {
	path, env := testEnv(t)
	defer os.RemoveAll(path)
	items := []simpleitem{
		{"k0", "v0"},
		{"k1", "v1"},
		{"k2", "v2"},
		{"k3", "v3"},
		{"k4", "v4"},
		{"k5", "v5"},
	}
	loadTestData(t, env, items)
	for i, test := range []struct {
		filtered []simpleitem
		errcheck
	}{
		{
			items,
			nil,
		},
	} {
		filtered, err := simplescan(env)
		if err != nil {
			t.Errorf("test %d: %v", i, err)
		}
		if !reflect.DeepEqual(filtered, test.filtered) {
			t.Errorf("test %d: unexpected items %q (!= %q)", i, filtered, test.filtered)
		}
	}
}

func TestScanner_Set(t *testing.T) {
	path, env := testEnv(t)
	defer os.RemoveAll(path)
	items := []simpleitem{
		{"k0", "v0"},
		{"k1", "v1"},
		{"k2", "v2"},
		{"k3", "v3"},
		{"k4", "v4"},
		{"k5", "v5"},
	}
	loadTestData(t, env, items)
	var tail []simpleitem
	err := env.View(func(txn *lmdb.Txn) (err error) {
		dbi, err := txn.OpenRoot(0)
		if err != nil {
			return err
		}
		s := New(txn, dbi)
		defer s.Close()

		s.Set([]byte("k34"), nil, lmdb.SetRange)
		tail, err = remaining(s)
		return err
	})
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(tail, items[4:]) {
		t.Errorf("items: %q (!= %q)", tail, items)
	}
}

func TestScanner_SetNext(t *testing.T) {
	path, env := testEnv(t)
	defer os.RemoveAll(path)
	items := []simpleitem{
		{"k0", "v0"},
		{"k1", "v1"},
		{"k2", "v2"},
		{"k3", "v3"},
		{"k4", "v4"},
		{"k5", "v5"},
	}
	loadTestData(t, env, items)
	var head []simpleitem
	err := env.View(func(txn *lmdb.Txn) (err error) {
		dbi, err := txn.OpenRoot(0)
		if err != nil {
			return err
		}
		s := New(txn, dbi)
		defer s.Close()

		s.SetNext([]byte("k34"), nil, lmdb.SetRange, lmdb.Prev)
		head, err = remaining(s)
		return err
	})
	if err != nil {
		t.Error(err)
	}

	// reverse head before testing its value
	n := len(head)
	for i := 0; i < n/2; i++ {
		head[i], head[n-1-i] = head[n-1-i], head[i]
	}

	if !reflect.DeepEqual(head, items[:5]) {
		t.Errorf("items: %q (!= %q)", head, items)
	}
}

type simpleitem [2]string

func loadTestData(t *testing.T, env *lmdb.Env, items []simpleitem) {
	err := env.Update(func(txn *lmdb.Txn) (err error) {
		db, err := txn.OpenRoot(0)
		if err != nil {
			return err
		}
		for _, item := range items {
			err := txn.Put(db, []byte(item[0]), []byte(item[1]), 0)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

func simplescan(env *lmdb.Env) (items []simpleitem, err error) {
	err = env.View(func(txn *lmdb.Txn) (err error) {
		db, err := txn.OpenRoot(0)
		if err != nil {
			return err
		}

		s := New(txn, db)
		defer s.Close()

		items, err = remaining(s)
		return err
	})
	return items, err
}

func remaining(s *Scanner) (items []simpleitem, err error) {
	for s.Scan() {
		items = append(items, simpleitem{string(s.Key()), string(s.Val())})
	}
	err = s.Err()
	if err != nil {
		return nil, err
	}
	return items, nil
}

func testEnv(t *testing.T) (path string, env *lmdb.Env) {
	dir, err := ioutil.TempDir("", "test-lmdb-env-")
	if err != nil {
		t.Fatal(err)
	}
	cleanAndDie := func() {
		os.RemoveAll(dir)
		t.Fatal(err)
	}

	env, err = lmdb.NewEnv()
	if err != nil {
		cleanAndDie()
	}
	err = env.Open(dir, 0, 0644)
	if err != nil {
		cleanAndDie()
	}

	return dir, env
}
