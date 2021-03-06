package bucket

import (
	"fmt"
	"math/rand"
	"os"
	"testing"

	"encoding/json"

	"bytes"

	"github.com/boltdb/bolt"
	"github.com/stretchr/testify/assert"
)

type person struct {
	Name string
	Age  int
}

func prepareDB(t *testing.T) (*bolt.DB, func()) {
	f := fmt.Sprintf("test%d.db", rand.Intn(1024))
	db, err := bolt.Open(f, 0700, nil)
	assert.Nil(t, err)
	return db, func() {
		db.Close()
		os.Remove(f)
	}
}

func TestBktUpdate(t *testing.T) {
	testCases := []struct {
		Init      map[string]person
		UpdateAge map[string]int
	}{
		{
			map[string]person{
				"1": person{"XiaoHei", 10},
				"2": person{"XiaoHuang", 11},
			},
			map[string]int{
				"1": 20,
				"2": 21,
			},
		},
	}

	db, cancel := prepareDB(t)
	defer cancel()

	for _, tc := range testCases {
		bkt, err := New([]byte(fmt.Sprintf("bkt%d", rand.Int31n(1024))), db)
		assert.Nil(t, err)
		// init value
		for k, v := range tc.Init {
			d, err := json.Marshal(v)
			assert.Nil(t, err)
			bkt.Put([]byte(k), d)
		}

		// update value
		for k, v := range tc.UpdateAge {
			err := bkt.Update([]byte(k), func(val []byte) ([]byte, error) {
				var p person
				if err := json.NewDecoder(bytes.NewReader(val)).Decode(&p); err != nil {
					return nil, err
				}
				p.Age = v
				d, err := json.Marshal(p)
				if err != nil {
					return nil, err
				}
				return d, nil
			})
			assert.Nil(t, err)
		}

		// check the updated value
		for k, v := range tc.UpdateAge {
			val := bkt.Get([]byte(k))
			var p person
			err := json.NewDecoder(bytes.NewReader(val)).Decode(&p)
			assert.Nil(t, err)
			assert.Equal(t, v, p.Age)
		}
	}
}

func TestDelete(t *testing.T) {
	testCases := []struct {
		Name string
		Init map[string]string
		Del  string
		Err  error
	}{
		{
			"Delete exist",
			map[string]string{
				"a": "1",
				"b": "2",
			},
			"a",
			nil,
		},
		{
			"Delete none exist",
			map[string]string{
				"a": "1",
			},
			"b",
			nil,
		},
	}
	db, cancel := prepareDB(t)
	defer cancel()
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			bkt, err := New([]byte(fmt.Sprintf("abc%d", rand.Int31n(1024))), db)
			assert.Nil(t, err)
			for k, v := range tc.Init {
				err := bkt.Put([]byte(k), []byte(v))
				assert.Nil(t, err)
			}

			err = bkt.Delete([]byte(tc.Del))
			assert.Equal(t, tc.Err, err)

			// check if this value is deleted
			v := bkt.Get([]byte(tc.Del))
			assert.Nil(t, v)
		})
	}
}

func TestGetAll(t *testing.T) {
	testCases := []struct {
		init map[string]string
	}{
		{
			map[string]string{
				"a": "1",
				"b": "2",
				"c": "3",
			},
		},
	}
	db, cancel := prepareDB(t)
	defer cancel()

	for _, tc := range testCases {
		bkt, err := New([]byte(fmt.Sprintf("abc%d", rand.Int31n(1024))), db)
		assert.Nil(t, err)
		// init bkt
		for k, v := range tc.init {
			bkt.Put([]byte(k), []byte(v))
		}

		// get all
		vs := bkt.GetAll()
		for k, v := range vs {
			assert.Equal(t, string(v), tc.init[k.(string)])
		}
	}
}

func TestRangeUpdate(t *testing.T) {
	testCases := []struct {
		init map[string]string
		up   map[string]string
	}{
		{
			map[string]string{
				"a": "1",
				"b": "2",
				"c": "3",
			},
			map[string]string{
				"a": "10",
				"b": "20",
				"c": "30",
			},
		},
	}
	db, cancel := prepareDB(t)
	defer cancel()

	for _, tc := range testCases {
		bkt, err := New([]byte(fmt.Sprintf("asd%d", rand.Int31n(1024))), db)
		assert.Nil(t, err)
		for k, v := range tc.init {
			bkt.Put([]byte(k), []byte(v))
		}

		// range update
		bkt.RangeUpdate(func(k, v []byte) ([]byte, error) {
			return []byte(tc.up[string(k)]), nil
		})

		// check if the value has been updated
		for k, v := range tc.up {
			assert.Equal(t, []byte(v), bkt.Get([]byte(k)))
		}
	}
}

func TestIsExsit(t *testing.T) {
	testCases := []struct {
		init  map[string]string
		k     string
		exist bool
	}{
		{
			map[string]string{
				"a": "1",
				"b": "2",
			},
			"a",
			true,
		},
		{
			map[string]string{
				"a": "1",
				"b": "2",
			},
			"b",
			true,
		},
		{
			map[string]string{
				"a": "1",
				"b": "2",
			},
			"c",
			false,
		},
		{
			map[string]string{},
			"c",
			false,
		},
	}

	db, cancel := prepareDB(t)
	defer cancel()

	for _, tc := range testCases {
		bkt, err := New([]byte(fmt.Sprintf("asdf%d", rand.Int31n(1024))), db)
		assert.Nil(t, err)

		// init the bucket
		for k, v := range tc.init {
			bkt.Put([]byte(k), []byte(v))
		}

		assert.Equal(t, tc.exist, bkt.IsExist([]byte(tc.k)))
	}
}

func TestForEach(t *testing.T) {
	testCases := []struct {
		init map[string]string
	}{
		{
			map[string]string{
				"a": "1",
				"b": "2",
				"c": "3",
			},
		},
		{
			map[string]string{},
		},
	}
	db, cancel := prepareDB(t)
	defer cancel()
	for _, tc := range testCases {
		bkt, err := New([]byte(fmt.Sprintf("fasd%d", rand.Int31n(1024))), db)
		assert.Nil(t, err)
		for k, v := range tc.init {
			bkt.Put([]byte(k), []byte(v))
		}

		var count int
		bkt.ForEach(func(k, v []byte) error {
			count++
			assert.Equal(t, string(v), tc.init[string(k)])
			return nil
		})

		assert.Equal(t, len(tc.init), count)
	}
}

func TestLen(t *testing.T) {
	testCases := []struct {
		data map[string]string
		len  int
	}{
		{
			map[string]string{},
			0,
		},
		{
			map[string]string{
				"a": "1",
			},
			1,
		},
		{
			map[string]string{
				"a": "1",
				"b": "2",
				"c": "3",
				"d": "4",
			},
			4,
		},
	}

	db, cl := prepareDB(t)
	defer cl()
	for _, tc := range testCases {
		bkt, err := New([]byte(fmt.Sprintf("adsf%d", rand.Int31n(1024))), db)
		assert.Nil(t, err)
		for k, v := range tc.data {
			bkt.Put([]byte(k), []byte(v))
		}

		assert.Equal(t, tc.len, bkt.Len())
	}
}
