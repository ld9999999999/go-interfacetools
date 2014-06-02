// BSD licensed

package interfacetools

import (
	"encoding/json"
	"log"
	"testing"
	"fmt"
)

type MStruct_o struct {
	V string  `json:"v"`
}

// Example implementaiton of CopyIn custom converter
func (m *MStruct_o) CopyIn(v interface{}) error {
	if x, ok := v.(map[string] interface{}); ok {
		if n, ok := x["v"]; ok {
			m.V = fmt.Sprintf("copy in of different source type: %v", n)
		}
	}
	return nil
}

func TestCopyTo(t *testing.T) {
	type MStruct struct {
		V int    `json:"v"`
		Nested map[string] string `json:"nested"`
	}

	newM := func(i int, k, v string) *MStruct {
		return &MStruct{i, map[string]string{k:v}}
	}

	type TStruct struct {
		B  bool    `json:"b"`
		I  string  `json:"i"`
		J  int     `json:"j"`
		JP *int    `json:"jp"`
		K  float64 `json:"k"`
		M0 map[string] *MStruct `json:"m0"`
		M1 map[string] MStruct  `json:"m1"`
		S  []int   `json:"s"`
		SP []*int  `json:"sp"`
		X  interface{} `json:"x"`
		Z  *int    `json:"z"`
		ByName string
	}

	type TStruct_nested struct {
		TStruct
		ParentI string `json:"i"`
	}

	// structs used for copyout with different types to original
	type TStruct_o struct {
		I string       `json:"i"`
		J int          `json:"j"`
		K float64      `json:"k"`
		M0 map[string] *MStruct_o `json:"m0"`
		M1 map[string] MStruct    `json:"m1"`
		S []int        `json:"s"`
		X interface{}  `json:"x"`
		Missing string `json:"missing"`
	}

	// Start of test code

	var ts TStruct = TStruct {
		B: true,
		I: "This is I",
		J: 101,
		K: 3.14,
		M0 : map[string] *MStruct{"a":newM(0,"one","two"), "b":newM(1,"three","four")},
		M1 : map[string] MStruct{"x":*newM(10,"alpha","beta"), "y":*newM(11,"gamma","delta")},
		S : []int{2,4,6,8},
		X : interface{}(map[string] string{"mx":"abc"}),
		Z : nil,
		ByName: "This is byname",
	}
	ts.JP = new(int)
	*ts.JP = 369
	nv := []int{101, 102, 103, 104}
	ts.SP = make([]*int, len(nv))
	for i := range nv {
		ts.SP[i] = &nv[i]
	}

	mbuf, err := json.Marshal(&ts)
	if err != nil {
		t.Fatalf("Marshal error: %s", err)
	}
	log.Println("mbuf:", string(mbuf))

	// The source json <map>
	var sj interface{}
	err = json.Unmarshal(mbuf, &sj)
	if err != nil {
		t.Fatalf("Unmarshal error:", err)
	}


	// Copy out to the same struct type
	var xs TStruct
	err = CopyOut(sj, &xs)
	if err != nil {
		t.Fatalf("CopyOut error: %s", err)
	}

	jsons, err := json.MarshalIndent(&xs, "", "  ")
	if err != nil {
		t.Fatalf("Marshal error: %s", err)
	}
	log.Println("CopyOut result 1.a:", string(jsons))


	// Copy out to nested anonymous struct
	var xs_nested TStruct_nested
	err = CopyOut(sj, &xs_nested)
	if err != nil {
		t.Fatalf("CopyOut error: %s", err)
	}

	jsons, err = json.MarshalIndent(&xs_nested, "", "  ")
	if err != nil {
		t.Fatalf("Marshal error: %s", err)
	}
	log.Println("CopyOut result 1.b:", string(jsons),
	            "\nParentI:", xs_nested.ParentI,
	            "\nChild-I:", xs_nested.TStruct.I)


	// Copy out to a struct with incompatible field types
	var xs0 TStruct_o
	err = CopyOut(sj, &xs0)
	if err != nil {
		t.Fatalf("CopyOut for struct with CopyIn() error: %s", err)
	}
	jsons, err = json.MarshalIndent(&xs0, "", "  ")
	log.Println("CopyOut result 2:", string(jsons))


	// Copy to a scalar
	var sj2 interface{}
	err = json.Unmarshal([]byte(`3`), &sj2)
	var n int
	err = CopyOut(sj2, &n)
	if err != nil {
		t.Fatalf("CopyOut scalar error: %s", err)
	}
	log.Println("N:", n)
}
