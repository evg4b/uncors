package fakedata

import (
	"sync"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/samber/lo"
)

var disablesFunctions = []string{
	// file
	"csv",
	"json",
	"xml",
	"fileextension",
	"filemimetype",

	// template
	"template",
	"markdown",
	"email_text",
	"fixed_width",

	// product
	"product",
	"productname",
	"productdescription",
	"productcategory",
	"productfeature",
	"productmaterial",

	// person
	"person",
	"teams",

	// generate
	"map",
	"generate",
	"regex",

	// address
	"address",

	// game
	"gamertag",
	"dice",

	// beer
	"beeralcohol",
	"beerblg",
	"beerhop",
	"beeribu",
	"beermalt",
	"beername",
	"beerstyle",
	"beeryeast",

	// car
	"car",
	"carmaker",
	"carmodel",
	"cartype",
	"carfueltype",
	"cartransmissiontype",

	// foods
	"fruit",
	"vegetable",
	"breakfast",
	"lunch",
	"dinner",
	"snack",
	"dessert",

	// misc
	"weighted",
	"flipacoin",

	// colors
	"safecolor",
	"nicecolors",

	// images
	"imagejpeg",
	"imagepng",

	// html
	"inputname",
	"svg",

	// payment
	"creditcard",
	"currency",

	// company
	"job",

	// hacker
	"hackerabbreviation",
	"hackeradjective",
	"hackeringverb",
	"hackernoun",
	"hackerphrase",
	"hackerverb",

	// hipster
	"hipsterword",
	"hipstersentence",
	"hipsterparagraph",

	// app
	"appname",
	"appversion",
	"appauthor",

	// animal
	"petname",
	"animal",
	"animaltype",
	"farmanimal",
	"cat",
	"dog",
	"bird",

	// language
	"programminglanguage",

	// number
	"shuffleints",
	"randomint",
	"hexuint",

	// string
	"lexify",
	"numerify",
	"shufflestrings",
	"randomstring",

	// celebrity
	"celebrityactor",
	"celebritybusiness",
	"celebritysport",

	// minecraft
	"minecraftore",
	"minecraftwood",
	"minecraftarmortier",
	"minecraftarmorpart",
	"minecraftweapon",
	"minecrafttool",
	"minecraftdye",
	"minecraftfood",
	"minecraftanimal",
	"minecraftvillagerjob",
	"minecraftvillagerstation",
	"minecraftvillagerlevel",
	"minecraftmobpassive",
	"minecraftmobneutral",
	"minecraftmobhostile",
	"minecraftmobboss",
	"minecraftbiome",
	"minecraftweather",

	// book
	"book",
	"booktitle",
	"bookauthor",
	"bookgenre",

	// movie
	"movie",
	"moviename",
	"moviegenre",

	// error
	"error",
	"errordatabase",
	"errorgrpc",
	"errorhttp",
	"errorhttpclient",
	"errorhttpserver",
	"errorruntime",

	// school
	"school",

	// other
	"comment",
	"drink",
	"errorobject",
	"errorvalidation",
	"interjection",
	"intrange",
	"languagebcp",
	"latituderange",
	"longituderange",
	"noundeterminer",
	"nounproper",
	"phraseadverb",
	"phrasenoun",
	"phrasepreposition",
	"phraseverb",
	"productupc",
	"pronounindefinite",
	"randomuint",
	"sentencesimple",
	"sql",
	"uintrange",
	"verbintransitive",
	"verbtransitive",
	"vowel",
}

var initPackage = sync.OnceFunc(func() {
	info := gofakeit.GetFuncLookup("loremipsumsentence")
	gofakeit.AddFuncLookup("string", gofakeit.Info{
		Display:     "Lorem Ipsum String",
		Category:    "string",
		Description: info.Description,
		Example:     info.Example,
		Output:      info.Output,
		Params:      info.Params,
		Generate:    info.Generate,
	})

	lo.ForEach(disablesFunctions, func(item string, _ int) {
		if gofakeit.GetFuncLookup(item) == nil {
			panic(item)
		}

		gofakeit.RemoveFuncLookup(item)
	})
})

func GetTypes() []string {
	initPackage()

	types := lo.Keys(gofakeit.FuncLookups)
	types = append(types, "object")
	types = append(types, "array")

	return types
}
