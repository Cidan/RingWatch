package main

import (
	"bufio"
	"embed"
	"strconv"
	"strings"
)

// Paramdex ER/Names tables (id<space>English name), vendored for reproducible,
// offline generation. Master branch, so DLC items are included.
//
//go:embed names/*.txt
var nameFS embed.FS

func loadNames() (nameMaps, error) {
	var nm nameMaps
	var err error
	if nm.weapon, err = loadNameMap("EquipParamWeapon"); err != nil {
		return nm, err
	}
	if nm.protector, err = loadNameMap("EquipParamProtector"); err != nil {
		return nm, err
	}
	if nm.accessory, err = loadNameMap("EquipParamAccessory"); err != nil {
		return nm, err
	}
	if nm.goods, err = loadNameMap("EquipParamGoods"); err != nil {
		return nm, err
	}
	if nm.gem, err = loadNameMap("EquipParamGem"); err != nil {
		return nm, err
	}
	return nm, nil
}

func loadNameMap(param string) (map[uint32]string, error) {
	data, err := nameFS.ReadFile("names/" + param + ".txt")
	if err != nil {
		return nil, err
	}
	m := map[uint32]string{}
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		id, rest, ok := strings.Cut(line, " ")
		if !ok {
			continue
		}
		n, err := strconv.ParseUint(strings.TrimSpace(id), 10, 32)
		if err != nil {
			continue
		}
		m[uint32(n)] = strings.TrimSpace(rest)
	}
	return m, sc.Err()
}
