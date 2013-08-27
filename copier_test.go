// BSD licensed

package interfacetools

import (
	"encoding/json"
	"log"
	"testing"
)

type MStruct struct {
	V int    `json:"v"`
}

type TStruct struct {
	B bool    `json:"b"`
	I string  `json:"i"`
	J int     `json:"j"`
	JP *int   `json:"jp"`
	K float64 `json:"k"`
	M0 map[string] *MStruct `json:"m0"`
	M1 map[string] MStruct  `json:"m1"`
	S []int  `json:"s"`
	X interface{} `json:"x"`
	Z *int   `json:"z"`
	ByName string
}

type MStruct_o struct {
	V string  `json:"v"`
}

type TStruct_o struct {
	I string  `json:"i"`
	J int     `json:"j"`
	K float64 `json:"k"`
	M0 map[string] *MStruct_o `json:"m0"`
	M1 map[string] MStruct  `json:"m1"`
	S []int  `json:"s"`
	X interface{} `json:"x"`
	Missing string `json:"missing"`
}

func TestDecoder(t *testing.T) {
	var ts TStruct = TStruct {
		B: true,
		I: "This is I",
		J: 101,
		K: 3.14,
		M0 : map[string] *MStruct{"a":&MStruct{0}, "b":&MStruct{1}},
		M1 : map[string] MStruct{"x":MStruct{10}, "y":MStruct{11}},
		S : []int{2,4,6,8},
		X : interface{}(map[string] string{"mx":"abc"}),
		Z : nil,
		ByName: "This is byname",
	}
	ts.JP = new(int)
	*ts.JP = 369

	mbuf, err := json.Marshal(&ts)
	log.Println("mbuf:", string(mbuf), err)

	var sj interface{}
	err = json.Unmarshal(mbuf, &sj)
	if err != nil {
		log.Println("Error:", err)
	}

	var xs TStruct
	err = CopyOut(sj, &xs)
	log.Println("ERROR DECODING:", err)

	xb, err := json.MarshalIndent(&xs, "", "  ")
	log.Println(">>XS:", string(xb))

	var xs0 TStruct_o
	err = CopyOut(sj, &xs0)
	log.Println("ERROR DECODING(expected):", err)
	xb, err = json.MarshalIndent(&xs0, "", "  ")
	log.Println(">>XS2:", string(xb))


	// Copy to a scalar
	var sj2 interface{}
	err = json.Unmarshal([]byte(`3`), &sj2)
	var n int
	err = CopyOut(sj2, &n)
	log.Println("N:", n, err)
}
