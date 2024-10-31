package medName

import (
	"unicode"
)

type words struct {
	ToRemove []string
}

var unwantedWords = map[string]struct{}{
	"اقراص": {}, "قرص": {}, "كبسول": {}, "شراب": {}, "شرب": {}, "لبوس": {},
	"حقن": {}, "مرهم": {}, "جل": {}, "جيل": {}, "كريم": {}, "محلول": {},
	"بودرة": {}, "بودره": {}, "قطرة": {}, "قطره": {}, "نقط": {}, "بخاخ": {},
	"سبراى": {}, "اسبراي": {}, "شرائط": {}, "لصقة": {}, "فيال": {}, "امبول": {},
	"امبولات": {}, "اكياس": {}, "فوار": {}, "حبيبات": {}, "سولوستار": {},
	"شامبو": {}, "غرغرة": {}, "لوسيون": {}, "لوشن": {}, "زيت": {}, "استنشاق": {},
	"شريط": {}, "للحقن": {}, "وحدة": {}, "دولية": {}, "دوليه": {}, "جرام": {},
	"جم": {}, "مجم": {}, "صابون": {}, "لبن": {}, "م": {}, "ملجم": {}, "ج": {},
	"مل": {}, "ملل": {}, "مللى": {}, "لتر": {}, "كبير": {}, "صغير": {}, "كيس": {},
	"زجاجه": {}, "سائل": {}, "ق": {}, "س": {}, "جديد": {}, "قديم": {}, "tab": {}, "سعر": {}, "كبسولة": {},
	"tablet": {}, "cap": {}, "capsule": {},
}

var arabicRange = &unicode.RangeTable{
	R16: []unicode.Range16{
		{Lo: 0x0600, Hi: 0x06FF, Stride: 1},
		{Lo: 0x0750, Hi: 0x077F, Stride: 1},
		{Lo: 0x08A0, Hi: 0x08FF, Stride: 1},
		{Lo: 0xFB50, Hi: 0xFDFF, Stride: 1},
		{Lo: 0xFE70, Hi: 0xFEFF, Stride: 1},
	},
}

type charType string

const (
	symbolType charType = "symbol"
	letterType charType = "letter"
	numberType charType = "number"
)