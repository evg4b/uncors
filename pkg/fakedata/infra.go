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
	"name",
	"nameprefix",
	"namesuffix",
	"firstname",
	"middlename",
	"lastname",
	"gender",
	"ssn",
	"hobby",
	// "contact", - unknown type
	"email",
	"phone",
	"phoneformatted",
	"teams",

	// generate
	// "struct",
	// "slice",
	"map",
	"generate",
	"regex",

	// auth
	"username",
	"password",

	// address
	"address",
	"city",
	"country",
	"countryabr",
	"state",
	"stateabr",
	"street",
	"streetname",
	"streetnumber",
	"streetprefix",
	"streetsuffix",
	"zip",
	"latitude",
	// "latitudeinrange",
	"longitude",
	// "longitudeinrange",

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

	// nouns
	"noun",
	"nouncommon",
	"nounconcrete",
	"nounabstract",
	"nouncollectivepeople",
	"nouncollectiveanimal",
	"nouncollectivething",
	"nouncountable",
	"noununcountable",

	// verbs
	"verb",
	"verbaction",
	"verblinking",
	"verbhelping",

	// adverbs
	"adverb",
	"adverbmanner",
	"adverbdegree",
	"adverbplace",
	"adverbtimedefinite",
	"adverbtimeindefinite",
	"adverbfrequencydefinite",
	"adverbfrequencyindefinite",

	// propositions
	"preposition",
	"prepositionsimple",
	"prepositiondouble",
	"prepositioncompound",

	// adjectives
	"adjective",
	"adjectivedescriptive",
	"adjectivequantitative",
	"adjectiveproper",
	"adjectivedemonstrative",
	"adjectivepossessive",
	"adjectiveinterrogative",
	"adjectiveindefinite",

	// pronouns
	"pronoun",
	"pronounpersonal",
	"pronounobject",
	"pronounpossessive",
	"pronounreflective",
	"pronoundemonstrative",
	"pronouninterrogative",
	"pronounrelative",

	// connectives
	"connective",
	"connectivetime",
	"connectivecomparative",
	"connectivecomplaint",
	"connectivelisting",
	"connectivecasual",
	"connectiveexamplify",

	// words
	"word",

	// sentences
	"sentence",
	"paragraph",
	"loremipsumword",
	"loremipsumsentence",
	"loremipsumparagraph",
	"question",
	"quote",
	"phrase",

	// foods
	"fruit",
	"vegetable",
	"breakfast",
	"lunch",
	"dinner",
	"snack",
	"dessert",

	// misc
	"bool",
	"uuid",
	"weighted",
	"flipacoin",
	// "randommapkey",
	// "shuffleanyslice",

	// colors
	"color",
	"hexcolor",
	"rgbcolor",
	"safecolor",
	"nicecolors",

	// images
	// "image",
	"imagejpeg",
	"imagepng",

	// internet
	"url",
	"domainname",
	"domainsuffix",
	"ipv4address",
	"ipv6address",
	"macaddress",
	"httpstatuscode",
	"httpstatuscodesimple",
	"loglevel",
	"httpmethod",
	"httpversion",
	"useragent",
	"chromeuseragent",
	"firefoxuseragent",
	"operauseragent",
	"safariuseragent",

	// html
	"inputname",
	"svg",

	// date/time
	"date",
	"pastdate",
	"futuredate",
	"daterange",
	"nanosecond",
	"second",
	"minute",
	"hour",
	"month",
	"monthstring",
	"day",
	"weekday",
	"year",
	"timezone",
	"timezoneabv",
	"timezonefull",
	"timezoneoffset",
	"timezoneregion",

	// payment
	"price",
	"creditcard",
	"creditcardcvv",
	"creditcardexp",
	"creditcardnumber",
	"creditcardtype",
	"currency",
	"currencylong",
	"currencyshort",
	"achrouting",
	"achaccount",
	"bitcoinaddress",
	"bitcoinprivatekey",

	// finance
	"cusip",
	"isin",

	// company
	"bs",
	"blurb",
	"buzzword",
	"company",
	"companysuffix",
	"job",
	"jobdescriptor",
	"joblevel",
	"jobtitle",
	"slogan",

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

	// emoji
	"emoji",
	"emojidescription",
	"emojicategory",
	"emojialias",
	"emojitag",

	// language
	"language",
	"languageabbreviation",
	"programminglanguage",
	// "programminglanguagebest",

	// number
	"number",
	"int",
	"intn",
	"int8",
	"int16",
	"int32",
	"int64",
	"uint",
	"uintn",
	"uint8",
	"uint16",
	"uint32",
	"uint64",
	"float32",
	"float32range",
	"float64",
	"float64range",
	"shuffleints",
	"randomint",
	"hexuint",

	// string
	"digit",
	"digitn",
	"letter",
	"lettern",
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
	// "errorinput",
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
