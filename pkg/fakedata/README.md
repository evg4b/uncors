# github.com/evg4b/uncors/pkg/fakedata

## Address

## City


Part of a country with significant population, often a central hub for culture and commerce


Return type: `string`


Example:


```
Marcelside
```

## Country


Nation with its own government and defined territory


Return type: `string`


Example:


```
United States of America
```

## Countryabr


Shortened 2-letter form of a country's name


Return type: `string`


Example:


```
US
```

## Latitude


Geographic coordinate specifying north-south position on Earth's surface


Return type: `float`


Example:


```
-73.534056
```

## Longitude


Geographic coordinate indicating east-west position on Earth's surface


Return type: `float`


Example:


```
-147.068112
```

## State


Governmental division within a country, often having its own laws and government


Return type: `string`


Example:


```
Illinois
```

## Stateabr


Shortened 2-letter form of a state or province


Return type: `string`


Example:


```
IL
```

## Street


Public road in a city or town, typically with houses and buildings on each side


Return type: `string`


Example:


```
364 East Rapidsborough
```

## Streetname


Name given to a specific road or street


Return type: `string`


Example:


```
View
```

## Streetnumber


Numerical identifier assigned to a street


Return type: `string`


Example:


```
13645
```

## Streetprefix


Directional or descriptive term preceding a street name (e.g., 'East', 'N')


Return type: `string`


Example:


```
East
```

## Streetsuffix


Designation at the end of a street name indicating type (e.g., 'Ave', 'St')


Return type: `string`


Example:


```
Ave
```

## Unit


Unit identifier within a building, such as apartment number, suite, or office


Return type: `string`


Example:


```
Apt 123
```

## Zip


Numerical code for postal address sorting, specific to a geographic area


Return type: `string`


Example:


```
13645
```

## Auth

## Password


Secret word or phrase used to authenticate access to a system or account


Return type: `string`


Options:


 - lower (bool) - Whether or not to add lower case characters
 - upper (bool) - Whether or not to add upper case characters
 - numeric (bool) - Whether or not to add numeric characters
 - special (bool) - Whether or not to add special characters
 - space (bool) - Whether or not to add spaces
 - length (int) - Number of characters in password


Example:


```
EEP+wwpk 4lU-eHNXlJZ4n K9%v&TZ9e
```

## Username


Unique identifier assigned to a user for accessing an account or system


Return type: `string`


Example:


```
Daniel1364
```

## Color

## Color


Hue seen by the eye, returns the name of the color like red or blue


Return type: `string`


Example:


```
MediumOrchid
```

## Hexcolor


Six-digit hexadecimal code representing a color in the RGB color model


Return type: `string`


Example:


```
#a99fb4
```

## Rgbcolor


Color defined by red, green, and blue light values


Return type: `[]int`


Example:


```
[85, 224, 195]
```

## Company

## Blurb


Brief description or summary of a company's purpose, products, or services


Return type: `string`


Example:


```
word
```

## Bs


Random bs company word


Return type: `string`


Example:


```
front-end
```

## Buzzword


Trendy or overused term often used in business to sound impressive


Return type: `string`


Example:


```
disintermediate
```

## Company


Designated official name of a business or organization


Return type: `string`


Example:


```
Moen, Pagac and Wuckert
```

## Companysuffix


Suffix at the end of a company name, indicating business structure, like 'Inc.' or 'LLC'


Return type: `string`


Example:


```
Inc
```

## Jobdescriptor


Word used to describe the duties, requirements, and nature of a job


Return type: `string`


Example:


```
Central
```

## Joblevel


Random job level


Return type: `string`


Example:


```
Assurance
```

## Jobtitle


Specific title for a position or role within a company or organization


Return type: `string`


Example:


```
Director
```

## Slogan


Catchphrase or motto used by a company to represent its brand or values


Return type: `string`


Example:


```
Universal seamless Focus, interactive.
```

## Datetime

## Date


Representation of a specific day, month, and year, often used for chronological reference


Return type: `string`


Example:


```
2006-01-02T15:04:05Z07:00
```

## Daterange


Random date between two ranges


Return type: `string`


Options:


 - startdate (string) - Start date time string
 - enddate (string) - End date time string
 - format (string) - Date time string format


Example:


```
1995-06-15T14:30:00Z
```

## Day


24-hour period equivalent to one rotation of Earth on its axis


Return type: `int`


Example:


```
12
```

## Futuredate


Date that has occurred after the current moment in time


Return type: `time`


Example:


```
2107-01-24 13:00:35.820738079 +0000 UTC
```

## Hour


Unit of time equal to 60 minutes


Return type: `int`


Example:


```
8
```

## Minute


Unit of time equal to 60 seconds


Return type: `int`


Example:


```
34
```

## Month


Division of the year, typically 30 or 31 days long


Return type: `string`


Example:


```
1
```

## Monthstring


String representation of a month name


Return type: `string`


Example:


```
September
```

## Nanosecond


Unit of time equal to one billionth (10^-9) of a second


Return type: `int`


Example:


```
196446360
```

## Pastdate


Date that has occurred before the current moment in time


Return type: `time`


Example:


```
2007-01-24 13:00:35.820738079 +0000 UTC
```

## Second


Unit of time equal to 1/60th of a minute


Return type: `int`


Example:


```
43
```

## Time


Random time string in the specified format


Return type: `string`


Example:


```
14:30:25
```

## Timerange


Random time string between start and end times


Return type: `string`


Options:


 - starttime (string) - Start time string in the specified format
 - endtime (string) - End time string in the specified format
 - format (string) - Time format string. Supports Java time format patterns or Go time format patterns


Example:


```
10:15:30
```

## Timezone


Region where the same standard time is used, based on longitudinal divisions of the Earth


Return type: `string`


Example:


```
Kaliningrad Standard Time
```

## Timezoneabv


Abbreviated 3-letter word of a timezone


Return type: `string`


Example:


```
KST
```

## Timezonefull


Full name of a timezone


Return type: `string`


Example:


```
(UTC+03:00) Kaliningrad, Minsk
```

## Timezoneoffset


The difference in hours from Coordinated Universal Time (UTC) for a specific region


Return type: `float32`


Example:


```
-5
```

## Timezoneregion


Geographic area sharing the same standard time


Return type: `string`


Example:


```
America/Alaska
```

## Weekday


Day of the week excluding the weekend


Return type: `string`


Example:


```
Friday
```

## Year


Period of 365 days, the time Earth takes to orbit the Sun


Return type: `int`


Example:


```
1900
```

## Emoji

## Emoji


Digital symbol expressing feelings or ideas in text messages and online chats


Return type: `string`


Example:


```
🤣
```

## Emojialias


Alternative name or keyword used to represent a specific emoji in text or code


Return type: `string`


Example:


```
smile
```

## Emojicategory


Group or classification of emojis based on their common theme or use, like 'smileys' or 'animals'


Return type: `string`


Example:


```
Smileys & Emotion
```

## Emojidescription


Brief explanation of the meaning or emotion conveyed by an emoji


Return type: `string`


Example:


```
face vomiting
```

## Emojitag


Label or keyword associated with an emoji to categorize or search for it easily


Return type: `string`


Example:


```
happy
```

## Finance

## Cusip


Unique identifier for securities, especially bonds, in the United States and Canada


Return type: `string`


Example:


```
38259P508
```

## Isin


International standard code for uniquely identifying securities worldwide


Return type: `string`


Example:


```
CVLRQCZBXQ97
```

## Internet

## Chromeuseragent


The specific identification string sent by the Google Chrome web browser when making requests on the internet


Return type: `string`


Example:


```
Mozilla/5.0 (X11; Linux i686) AppleWebKit/5312 (KHTML, like Gecko) Chrome/39.0.836.0 Mobile Safari/5312
```

## Domainname


Human-readable web address used to identify websites on the internet


Return type: `string`


Example:


```
centraltarget.biz
```

## Domainsuffix


The part of a domain name that comes after the last dot, indicating its type or purpose


Return type: `string`


Example:


```
org
```

## Firefoxuseragent


The specific identification string sent by the Firefox web browser when making requests on the internet


Return type: `string`


Example:


```
Mozilla/5.0 (Macintosh; U; PPC Mac OS X 10_8_3 rv:7.0) Gecko/1900-07-01 Firefox/37.0
```

## Httpmethod


Verb used in HTTP requests to specify the desired action to be performed on a resource


Return type: `string`


Example:


```
HEAD
```

## Httpstatuscode


Random HTTP status code


Return type: `int`


Example:


```
200
```

## Httpstatuscodesimple


Three-digit number returned by a web server to indicate the outcome of an HTTP request


Return type: `int`


Example:


```
404
```

## Httpversion


Number indicating the version of the HTTP protocol used for communication between a client and a server


Return type: `string`


Example:


```
HTTP/1.1
```

## Ipv4address


Numerical label assigned to devices on a network for identification and communication


Return type: `string`


Example:


```
222.83.191.222
```

## Ipv6address


Numerical label assigned to devices on a network, providing a larger address space than IPv4 for internet communication


Return type: `string`


Example:


```
2001:cafe:8898:ee17:bc35:9064:5866:d019
```

## Loglevel


Classification used in logging to indicate the severity or priority of a log entry


Return type: `string`


Example:


```
error
```

## Macaddress


Unique identifier assigned to network interfaces, often used in Ethernet networks


Return type: `string`


Example:


```
cb:ce:06:94:22:e9
```

## Operauseragent


The specific identification string sent by the Opera web browser when making requests on the internet


Return type: `string`


Example:


```
Opera/8.39 (Macintosh; U; PPC Mac OS X 10_8_7; en-US) Presto/2.9.335 Version/10.00
```

## Safariuseragent


The specific identification string sent by the Safari web browser when making requests on the internet


Return type: `string`


Example:


```
Mozilla/5.0 (iPad; CPU OS 8_3_2 like Mac OS X; en-US) AppleWebKit/531.15.6 (KHTML, like Gecko) Version/4.0.5 Mobile/8B120 Safari/6531.15.6
```

## Url


Web address that specifies the location of a resource on the internet


Return type: `string`


Example:


```
http://www.principalproductize.biz/target
```

## Useragent


String sent by a web browser to identify itself when requesting web content


Return type: `string`


Example:


```
Mozilla/5.0 (Windows NT 5.0) AppleWebKit/5362 (KHTML, like Gecko) Chrome/37.0.834.0 Mobile Safari/5362
```

## Language

## Language


System of communication using symbols, words, and grammar to convey meaning between individuals


Return type: `string`


Example:


```
Kazakh
```

## Languageabbreviation


Shortened form of a language's name


Return type: `string`


Example:


```
kk
```

## Misc

## Bool


Data type that represents one of two possible values, typically true or false


Return type: `bool`


Example:


```
true
```

## Uuid


128-bit identifier used to uniquely identify objects or entities in computer systems


Return type: `string`


Example:


```
590c1440-9888-45b0-bd51-a817ee07c3f2
```

## Number

## Float32


Floating-point number with 32-bit single precision (IEEE 754)


Return type: `float32`


Example:


```
3.1128167e+37
```

## Float32range


Float32 value within a given range


Return type: `float32`


Options:


 - min (float) - Minimum float32 value
 - max (float) - Maximum float32 value


Example:


```
914774.6
```

## Float64


Floating-point number with 64-bit double precision (IEEE 754)


Return type: `float64`


Example:


```
1.644484108270445e+307
```

## Float64range


Float64 value within a given range


Return type: `float64`


Options:


 - min (float) - Minimum float64 value
 - max (float) - Maximum float64 value


Example:


```
914774.5585333086
```

## Int


Signed integer


Return type: `int`


Example:


```
14866
```

## Int16


Signed 16-bit integer, range −32,768–32,767


Return type: `int16`


Example:


```
2200
```

## Int32


Signed 32-bit integer, range −2,147,483,648–2,147,483,647


Return type: `int32`


Example:


```
-1072427943
```

## Int64


Signed 64-bit integer, range −9,223,372,036,854,775,808–9,223,372,036,854,775,807


Return type: `int64`


Example:


```
-8379641344161477543
```

## Int8


Signed 8-bit integer, range −128–127


Return type: `int8`


Example:


```
24
```

## Intn


Integer between 0 (inclusive) and n (exclusive)


Return type: `int`


Example:


```
32783
```

## Number


Integer used for counting or measuring, with optional bounds


Return type: `int`


Options:


 - min (int) - Minimum integer value
 - max (int) - Maximum integer value


Example:


```
14866
```

## Uint


Unsigned integer (nonnegative whole number)


Return type: `uint`


Example:


```
14866
```

## Uint16


Unsigned 16-bit integer, range 0–65,535


Return type: `uint16`


Example:


```
34968
```

## Uint32


Unsigned 32-bit integer, range 0–4,294,967,295


Return type: `uint32`


Example:


```
1075055705
```

## Uint64


Unsigned 64-bit integer, range 0–18,446,744,073,709,551,615


Return type: `uint64`


Example:


```
843730692693298265
```

## Uint8


Unsigned 8-bit integer, range 0–255


Return type: `uint8`


Example:


```
152
```

## Uintn


Unsigned integer between 0 (inclusive) and n (exclusive)


Return type: `uint`


Example:


```
32783
```

## Payment

## Achaccount


Bank account number used for Automated Clearing House transactions


Return type: `string`


Example:


```
491527954328
```

## Achrouting


Nine-digit code used in the U.S. for identifying a bank in ACH transactions


Return type: `string`


Example:


```
513715684
```

## Bankname


Name of a financial institution that offers banking services


Return type: `string`


Example:


```
Wells Fargo
```

## Banktype


Classification of a bank based on its services and operations


Return type: `string`


Example:


```
Investment Bank
```

## Bitcoinaddress


Cryptographic identifier used to send and receive Bitcoin


Return type: `string`


Example:


```
1BoatSLRHtKNngkdXEeobR76b53LETtpyT
```

## Bitcoinprivatekey


Secret key that allows access and control over Bitcoin holdings


Return type: `string`


Example:


```
5HueCGU8rMjxEXxiPuD5BDuG6o5xjA7QkbPp
```

## Creditcardcvv


Three or four-digit security code on a credit card


Return type: `string`


Example:


```
513
```

## Creditcardexp


Expiration date of a credit card


Return type: `string`


Example:


```
01/27
```

## Creditcardnumber


Unique number on a credit card used for electronic payments


Return type: `string`


Options:


 - types ([]string) - A select number of types you want to use when generating a credit card number
 - bins ([]string) - Optional list of prepended bin numbers to pick from
 - gaps (bool) - Whether or not to have gaps in number


Example:


```
4111111111111111
```

## Creditcardtype


Classification of credit cards based on the issuing company


Return type: `string`


Example:


```
Visa
```

## Currencylong


Complete name of a specific currency used in financial transactions


Return type: `string`


Example:


```
United States Dollar
```

## Currencyshort


Short 3-letter ISO code used to represent a specific currency


Return type: `string`


Example:


```
USD
```

## Price


The amount of money assigned to a product, service, or asset in a transaction


Return type: `float64`


Options:


 - min (float) - Minimum price value
 - max (float) - Maximum price value


Example:


```
92.26
```

## Person

## Ein


Nine-digit Employer Identification Number used by businesses for tax purposes


Return type: `string`


Example:


```
12-3456789
```

## Email


Electronic mail address


Return type: `string`


Example:


```
markusmoen@pagac.net
```

## Firstname


The name given to a person at birth


Return type: `string`


Example:


```
Markus
```

## Gender


Classification that identifies gender


Return type: `string`


Example:


```
male
```

## Hobby


An activity pursued for leisure and pleasure


Return type: `string`


Example:


```
Swimming
```

## Lastname


The family name or surname of an individual


Return type: `string`


Example:


```
Daniel
```

## Middlename


Name between a person's first name and last name


Return type: `string`


Example:


```
Belinda
```

## Name


The given and family name of an individual


Return type: `string`


Example:


```
Markus Moen
```

## Nameprefix


A title or honorific added before a person's name


Return type: `string`


Example:


```
Mr.
```

## Namesuffix


A title or designation added after a person's name


Return type: `string`


Example:


```
Jr.
```

## Phone


Numerical sequence used to contact individuals via telephone or mobile devices


Return type: `string`


Example:


```
6136459948
```

## Phoneformatted


Formatted phone number of a person


Return type: `string`


Example:


```
136-459-9489
```

## Ssn


Unique nine-digit identifier used for government and financial purposes in the United States


Return type: `string`


Example:


```
296446360
```

## Product

## Productisbn


ISBN-10 or ISBN-13 identifier for books


Return type: `string`


Example:


```
978-1-4028-9462-6
```

## Song

## Songartist


The artist of maker of song


Return type: `string`


Example:


```
Dua Lipa
```

## Songgenre


Category that classifies song based on common themes, styles, and storytelling approaches


Return type: `string`


Example:


```
Action
```

## Songname


Title or name of a specific song used for identification and reference


Return type: `string`


Example:


```
New Rules
```

## String

## Digit


Numerical symbol used to represent numbers


Return type: `string`


Example:


```
0
```

## Digitn


String of length N consisting of ASCII digits


Return type: `string`


Example:


```
0136459948
```

## Letter


Character or symbol from the American Standard Code for Information Interchange (ASCII) character set


Return type: `string`


Example:


```
g
```

## Lettern


ASCII string with length N


Return type: `string`


Example:


```
gbRMaRxHki
```

## String


Sentence of the Lorem Ipsum placeholder text used in design and publishing


Return type: `string`


Example:


```
Quia quae repellat consequatur quidem.
```

## Text

## Loremipsumparagraph


Paragraph of the Lorem Ipsum placeholder text used in design and publishing


Return type: `string`


Options:


 - paragraphcount (int) - Number of paragraphs
 - sentencecount (int) - Number of sentences in a paragraph
 - wordcount (int) - Number of words in a sentence
 - paragraphseparator (string) - String value to add between paragraphs


Example:


```
Quia quae repellat consequatur quidem nisi quo qui voluptatum accusantium quisquam amet. Quas et ut non dolorem ipsam aut enim assumenda mollitia harum ut. Dicta similique veniam nulla voluptas at excepturi non ad maxime at non. Eaque hic repellat praesentium voluptatem qui consequuntur dolor iusto autem velit aut. Fugit tempore exercitationem harum consequatur voluptatum modi minima aut eaque et et.

Aut ea voluptatem dignissimos expedita odit tempore quod aut beatae ipsam iste. Minus voluptatibus dolorem maiores eius sed nihil vel enim odio voluptatem accusamus. Natus quibusdam temporibus tenetur cumque sint necessitatibus dolorem ex ducimus iusto ex. Voluptatem neque dicta explicabo officiis et ducimus sit ut ut praesentium pariatur. Illum molestias nisi at dolore ut voluptatem accusantium et fugiat et ut.

Explicabo incidunt reprehenderit non quia dignissimos recusandae vitae soluta quia et quia. Aut veniam voluptas consequatur placeat sapiente non eveniet voluptatibus magni velit eum. Nobis vel repellendus sed est qui autem laudantium quidem quam ullam consequatur. Aut iusto ut commodi similique quae voluptatem atque qui fugiat eum aut. Quis distinctio consequatur voluptatem vel aliquid aut laborum facere officiis iure tempora.
```

## Loremipsumsentence


Sentence of the Lorem Ipsum placeholder text used in design and publishing


Return type: `string`


Example:


```
Quia quae repellat consequatur quidem.
```

## Paragraph


Distinct section of writing covering a single theme, composed of multiple sentences


Return type: `string`


Options:


 - paragraphcount (int) - Number of paragraphs
 - sentencecount (int) - Number of sentences in a paragraph
 - wordcount (int) - Number of words in a sentence
 - paragraphseparator (string) - String value to add between paragraphs


Example:


```
Interpret context record river mind press self should compare property outcome divide. Combine approach sustain consult discover explanation direct address church husband seek army. Begin own act welfare replace press suspect stay link place manchester specialist. Arrive price satisfy sign force application hair train provide basis right pay. Close mark teacher strengthen information attempt head touch aim iron tv take.
```

## Phrase


A small group of words standing together


Return type: `string`


Example:


```
time will tell
```

## Question


Statement formulated to inquire or seek clarification


Return type: `string`


Example:


```
Roof chia echo?
```

## Quote


Direct repetition of someone else's words


Return type: `string`


Example:


```
"Roof chia echo." - Lura Lockman
```

## Sentence


Set of words expressing a statement, question, exclamation, or command


Return type: `string`


Example:


```
Interpret context record river mind.
```

## Word

## Adjective


Word describing or modifying a noun


Return type: `string`


Example:


```
genuine
```

## Adjectivedemonstrative


Adjective used to point out specific things


Return type: `string`


Example:


```
this
```

## Adjectivedescriptive


Adjective that provides detailed characteristics about a noun


Return type: `string`


Example:


```
brave
```

## Adjectiveindefinite


Adjective describing a non-specific noun


Return type: `string`


Example:


```
few
```

## Adjectiveinterrogative


Adjective used to ask questions


Return type: `string`


Example:


```
what
```

## Adjectivepossessive


Adjective indicating ownership or possession


Return type: `string`


Example:


```
my
```

## Adjectiveproper


Adjective derived from a proper noun, often used to describe nationality or origin


Return type: `string`


Example:


```
Afghan
```

## Adjectivequantitative


Adjective that indicates the quantity or amount of something


Return type: `string`


Example:


```
a little
```

## Adverb


Word that modifies verbs, adjectives, or other adverbs


Return type: `string`


Example:


```
smoothly
```

## Adverbdegree


Adverb that indicates the degree or intensity of an action or adjective


Return type: `string`


Example:


```
intensely
```

## Adverbfrequencydefinite


Adverb that specifies how often an action occurs with a clear frequency


Return type: `string`


Example:


```
hourly
```

## Adverbfrequencyindefinite


Adverb that specifies how often an action occurs without specifying a particular frequency


Return type: `string`


Example:


```
occasionally
```

## Adverbmanner


Adverb that describes how an action is performed


Return type: `string`


Example:


```
stupidly
```

## Adverbplace


Adverb that indicates the location or direction of an action


Return type: `string`


Example:


```
east
```

## Adverbtimedefinite


Adverb that specifies the exact time an action occurs


Return type: `string`


Example:


```
now
```

## Adverbtimeindefinite


Adverb that gives a general or unspecified time frame


Return type: `string`


Example:


```
already
```

## Connective


Word used to connect words or sentences


Return type: `string`


Example:


```
such as
```

## Connectivecasual


Connective word used to indicate a cause-and-effect relationship between events or actions


Return type: `string`


Example:


```
an outcome of
```

## Connectivecomparative


Connective word used to indicate a comparison between two or more things


Return type: `string`


Example:


```
in addition
```

## Connectivecomplaint


Connective word used to express dissatisfaction or complaints about a situation


Return type: `string`


Example:


```
besides
```

## Connectiveexamplify


Connective word used to provide examples or illustrations of a concept or idea


Return type: `string`


Example:


```
then
```

## Connectivelisting


Connective word used to list or enumerate items or examples


Return type: `string`


Example:


```
firstly
```

## Connectivetime


Connective word used to indicate a temporal relationship between events or actions


Return type: `string`


Example:


```
finally
```

## Loremipsumword


Word of the Lorem Ipsum placeholder text used in design and publishing


Return type: `string`


Example:


```
quia
```

## Noun


Person, place, thing, or idea, named or referred to in a sentence


Return type: `string`


Example:


```
aunt
```

## Nounabstract


Ideas, qualities, or states that cannot be perceived with the five senses


Return type: `string`


Example:


```
confusion
```

## Nouncollectiveanimal


Group of animals, like a 'pack' of wolves or a 'flock' of birds


Return type: `string`


Example:


```
party
```

## Nouncollectivepeople


Group of people or things regarded as a unit


Return type: `string`


Example:


```
body
```

## Nouncollectivething


Group of objects or items, such as a 'bundle' of sticks or a 'cluster' of grapes


Return type: `string`


Example:


```
hand
```

## Nouncommon


General name for people, places, or things, not specific or unique


Return type: `string`


Example:


```
part
```

## Nounconcrete


Names for physical entities experienced through senses like sight, touch, smell, or taste


Return type: `string`


Example:


```
snowman
```

## Nouncountable


Items that can be counted individually


Return type: `string`


Example:


```
neck
```

## Noununcountable


Items that can't be counted individually


Return type: `string`


Example:


```
seafood
```

## Preposition


Words used to express the relationship of a noun or pronoun to other words in a sentence


Return type: `string`


Example:


```
other than
```

## Prepositioncompound


Preposition that can be formed by combining two or more prepositions


Return type: `string`


Example:


```
according to
```

## Prepositiondouble


Two-word combination preposition, indicating a complex relation


Return type: `string`


Example:


```
before
```

## Prepositionsimple


Single-word preposition showing relationships between 2 parts of a sentence


Return type: `string`


Example:


```
out
```

## Pronoun


Word used in place of a noun to avoid repetition


Return type: `string`


Example:


```
me
```

## Pronoundemonstrative


Pronoun that points out specific people or things


Return type: `string`


Example:


```
this
```

## Pronouninterrogative


Pronoun used to ask questions


Return type: `string`


Example:


```
what
```

## Pronounobject


Pronoun used as the object of a verb or preposition


Return type: `string`


Example:


```
it
```

## Pronounpersonal


Pronoun referring to a specific persons or things


Return type: `string`


Example:


```
it
```

## Pronounpossessive


Pronoun indicating ownership or belonging


Return type: `string`


Example:


```
mine
```

## Pronounreflective


Pronoun referring back to the subject of the sentence


Return type: `string`


Example:


```
myself
```

## Pronounrelative


Pronoun that introduces a clause, referring back to a noun or pronoun


Return type: `string`


Example:


```
as
```

## Verb


Word expressing an action, event or state


Return type: `string`


Example:


```
release
```

## Verbaction


Verb indicating a physical or mental action


Return type: `string`


Example:


```
close
```

## Verbhelping


Auxiliary verb that helps the main verb complete the sentence


Return type: `string`


Example:


```
be
```

## Verblinking


Verb that connects the subject of a sentence to a subject complement


Return type: `string`


Example:


```
was
```

## Word


Basic unit of language representing a concept or thing, consisting of letters and having meaning


Return type: `string`


Example:


```
man
```

