package fakedata_test

import (
	"sort"
	"testing"

	"github.com/evg4b/uncors/pkg/fakedata"
	"github.com/stretchr/testify/assert"
)

func TestGetTypes(t *testing.T) {
	expected := []string{
		// Inner types
		"object",
		"array",

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

		// string
		"digit",
		"digitn",
		"letter",
		"lettern",

		// sentences
		"sentence",
		"paragraph",
		"loremipsumword",
		"loremipsumsentence",
		"loremipsumparagraph",
		"question",
		"quote",
		"phrase",

		// misc
		"bool",
		"uuid",

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

		// emoji
		"emoji",
		"emojidescription",
		"emojicategory",
		"emojialias",
		"emojitag",

		// person
		"name",
		"nameprefix",
		"namesuffix",
		"firstname",
		"middlename",
		"lastname",
		"gender",
		"ssn",
		"hobby",
		"email",
		"phone",
		"phoneformatted",

		// auth
		"username",
		"password",

		// address
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
		"longitude",

		// colors
		"color",
		"hexcolor",
		"rgbcolor",

		// finance
		"cusip",
		"isin",

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

		// language
		"language",
		"languageabbreviation",

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

		// company
		"bs",
		"blurb",
		"buzzword",
		"company",
		"companysuffix",
		"jobdescriptor",
		"joblevel",
		"jobtitle",
		"slogan",

		// payment
		"price",
		"creditcardcvv",
		"creditcardexp",
		"creditcardnumber",
		"creditcardtype",
		"currencylong",
		"currencyshort",
		"achrouting",
		"achaccount",
		"bitcoinaddress",
		"bitcoinprivatekey",
	}

	actual := fakedata.GetTypes()

	sort.Strings(expected)
	sort.Strings(actual)

	assert.Equal(t, expected, actual)
}
