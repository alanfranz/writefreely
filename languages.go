/*
 * Copyright © 2026 Musing Studio LLC.
 *
 * This file is part of WriteFreely.
 *
 * WriteFreely is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License, included
 * in the LICENSE file in this source code package.
 */

package writefreely

import (
	"net/http"
	"strings"
)

// languageOption is a language that can be assigned to a post or blog.
type languageOption struct {
	Code string // ISO 639-1 code; posts.language is CHAR(2) on MySQL
	Name string // Native name of the language
	RTL  bool   // Whether the language is written right-to-left
}

// postLanguages lists the languages that can be assigned to a blog, in the
// same order (and with the same native names) as the language picker in the
// post metadata editor (templates/edit-meta.tmpl).
var postLanguages = []languageOption{
	{"ab", "аҧсуа бызшәа, аҧсшәа", false},
	{"aa", "Afaraf", false},
	{"af", "Afrikaans", false},
	{"ak", "Akan", false},
	{"sq", "Shqip", false},
	{"am", "አማርኛ", false},
	{"ar", "العربية", true},
	{"an", "aragonés", false},
	{"hy", "Հայերեն", false},
	{"as", "অসমীয়া", false},
	{"av", "авар мацӀ, магӀарул мацӀ", false},
	{"ae", "avesta", false},
	{"ay", "aymar aru", false},
	{"az", "azərbaycan dili", false},
	{"bm", "bamanankan", false},
	{"ba", "башҡорт теле", false},
	{"eu", "euskara, euskera", false},
	{"be", "беларуская мова", false},
	{"bn", "বাংলা", false},
	{"bh", "भोजपुरी", false},
	{"bi", "Bislama", false},
	{"bs", "bosanski jezik", false},
	{"br", "brezhoneg", false},
	{"bg", "български език", false},
	{"my", "ဗမာစာ", false},
	{"ca", "català", false},
	{"ch", "Chamoru", false},
	{"ce", "нохчийн мотт", false},
	{"ny", "chiCheŵa, chinyanja", false},
	{"zh", "中文 (Zhōngwén), 汉语, 漢語", false},
	{"cv", "чӑваш чӗлхи", false},
	{"kw", "Kernewek", false},
	{"co", "corsu, lingua corsa", false},
	{"cr", "ᓀᐦᐃᔭᐍᐏᐣ", false},
	{"hr", "hrvatski jezik", false},
	{"cs", "čeština, český jazyk", false},
	{"da", "dansk", false},
	{"dv", "ދިވެހި", true},
	{"nl", "Nederlands, Vlaams", false},
	{"dz", "རྫོང་ཁ", false},
	{"en", "English", false},
	{"eo", "Esperanto", false},
	{"et", "eesti, eesti keel", false},
	{"ee", "Eʋegbe", false},
	{"fo", "føroyskt", false},
	{"fj", "vosa Vakaviti", false},
	{"fi", "suomi, suomen kieli", false},
	{"fr", "français, langue française", false},
	{"ff", "Fulfulde, Pulaar, Pular", false},
	{"gl", "Galego", false},
	{"ka", "ქართული", false},
	{"de", "Deutsch", false},
	{"el", "ελληνικά", false},
	{"gn", "Avañe'ẽ", false},
	{"gu", "ગુજરાતી", false},
	{"ht", "Kreyòl ayisyen", false},
	{"ha", "(Hausa) هَوُسَ", true},
	{"he", "עברית", true},
	{"hz", "Otjiherero", false},
	{"hi", "हिन्दी, हिंदी", false},
	{"ho", "Hiri Motu", false},
	{"hu", "magyar", false},
	{"ia", "Interlingua", false},
	{"id", "Bahasa Indonesia", false},
	{"ie", "Interlingue", false},
	{"ga", "Gaeilge", false},
	{"ig", "Asụsụ Igbo", false},
	{"ik", "Iñupiaq, Iñupiatun", false},
	{"io", "Ido", false},
	{"is", "Íslenska", false},
	{"it", "Italiano", false},
	{"iu", "ᐃᓄᒃᑎᑐᑦ", false},
	{"ja", "日本語 (にほんご)", false},
	{"jv", "ꦧꦱꦗꦮ, Basa Jawa", false},
	{"kl", "kalaallisut, kalaallit oqaasii", false},
	{"kn", "ಕನ್ನಡ", false},
	{"kr", "Kanuri", false},
	{"ks", "कश्मीरी, كشميري‎", false},
	{"kk", "қазақ тілі", false},
	{"km", "ខ្មែរ, ខេមរភាសា, ភាសាខ្មែរ", false},
	{"ki", "Gĩkũyũ", false},
	{"rw", "Ikinyarwanda", false},
	{"ky", "Кыргызча, Кыргыз тили", false},
	{"kv", "коми кыв", false},
	{"kg", "Kikongo", false},
	{"ko", "한국어", false},
	{"ku", "Kurdî, كوردی‎", false},
	{"kj", "Kuanyama", false},
	{"la", "latine, lingua latina", false},
	{"lb", "Lëtzebuergesch", false},
	{"lg", "Luganda", false},
	{"li", "Limburgs", false},
	{"ln", "Lingála", false},
	{"lo", "ພາສາລາວ", false},
	{"lt", "lietuvių kalba", false},
	{"lu", "Kiluba", false},
	{"lv", "Latviešu Valoda", false},
	{"gv", "Gaelg, Gailck", false},
	{"mk", "македонски јазик", false},
	{"mg", "fiteny malagasy", false},
	{"ms", "Bahasa Melayu, بهاس ملايو‎", false},
	{"ml", "മലയാളം", false},
	{"mt", "Malti", false},
	{"mi", "te reo Māori", false},
	{"mr", "मराठी", false},
	{"mh", "Kajin M̧ajeļ", false},
	{"mn", "Монгол хэл", false},
	{"na", "Dorerin Naoero", false},
	{"nv", "Diné bizaad", false},
	{"nd", "isiNdebele", false},
	{"ne", "नेपाली", false},
	{"ng", "Owambo", false},
	{"nb", "Norsk Bokmål", false},
	{"nn", "Norsk Nynorsk", false},
	{"no", "Norsk", false},
	{"ii", "ꆈꌠ꒿ Nuosuhxop", false},
	{"nr", "isiNdebele", false},
	{"oc", "occitan, lenga d'òc", false},
	{"oj", "ᐊᓂᔑᓈᐯᒧᐎᓐ", false},
	{"cu", "ѩзыкъ словѣньскъ", false},
	{"om", "Afaan Oromoo", false},
	{"or", "ଓଡ଼ିଆ", false},
	{"os", "ирон æвзаг", false},
	{"pa", "ਪੰਜਾਬੀ", false},
	{"pi", "पाऴि", false},
	{"fa", "فارسی", true},
	{"pl", "Język Polski, Polszczyzna", false},
	{"ps", "پښتو", true},
	{"pt", "Português", false},
	{"qu", "Runa Simi, Kichwa", false},
	{"rm", "Rumantsch Grischun", false},
	{"rn", "Ikirundi", false},
	{"ro", "Română", false},
	{"ru", "Русский", false},
	{"sa", "संस्कृतम्", false},
	{"sc", "sardu", false},
	{"sd", "सिन्धी, سنڌي، سندھی‎", false},
	{"se", "Davvisámegiella", false},
	{"sm", "gagana fa'a Samoa", false},
	{"sg", "yângâ tî sängö", false},
	{"sr", "српски језик", false},
	{"gd", "Gàidhlig", false},
	{"sn", "chiShona", false},
	{"si", "සිංහල", false},
	{"sk", "Slovenčina, Slovenský Jazyk", false},
	{"sl", "Slovenski Jezik, Slovenščina", false},
	{"so", "Soomaaliga, af Soomaali", false},
	{"st", "Sesotho", false},
	{"es", "Español", false},
	{"su", "Basa Sunda", false},
	{"sw", "Kiswahili", false},
	{"ss", "SiSwati", false},
	{"sv", "Svenska", false},
	{"ta", "தமிழ்", false},
	{"te", "తెలుగు", false},
	{"tg", "тоҷикӣ, toçikī, تاجیکی‎", false},
	{"th", "ไทย", false},
	{"ti", "ትግርኛ", false},
	{"bo", "བོད་ཡིག", false},
	{"tk", "Türkmen, Түркмен", false},
	{"tl", "Wikang Tagalog", false},
	{"tn", "Setswana", false},
	{"to", "Faka Tonga", false},
	{"tr", "Türkçe", false},
	{"ts", "Xitsonga", false},
	{"tt", "татар теле, tatar tele", false},
	{"tw", "Twi", false},
	{"ty", "Reo Tahiti", false},
	{"ug", "ئۇيغۇرچە‎, Uyghurche", false},
	{"uk", "Українська", false},
	{"ur", "اردو", true},
	{"uz", "Oʻzbek, Ўзбек, أۇزبېك‎", false},
	{"ve", "Tshivenḓa", false},
	{"vi", "Tiếng Việt", false},
	{"vo", "Volapük", false},
	{"wa", "Walon", false},
	{"cy", "Cymraeg", false},
	{"wo", "Wollof", false},
	{"fy", "Frysk", false},
	{"xh", "isiXhosa", false},
	{"yi", "ייִדיש", true},
	{"yo", "Yorùbá", false},
	{"za", "Saɯ cueŋƅ, Saw cuengh", false},
	{"zu", "isiZulu", false},
}

var postLanguageCodes = func() map[string]bool {
	m := make(map[string]bool, len(postLanguages))
	for _, l := range postLanguages {
		m[l.Code] = true
	}
	return m
}()

// normalizeLanguage trims and lowercases the given language code and reports
// whether the result is a supported language. An empty value is valid, and
// means "no language".
func normalizeLanguage(lang string) (string, bool) {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if lang == "" {
		return "", true
	}
	return lang, postLanguageCodes[lang]
}

// setContentLanguage sets the Content-Language response header to the given
// language, if there is one.
func setContentLanguage(w http.ResponseWriter, lang string) {
	if lang != "" {
		w.Header().Set("Content-Language", lang)
	}
}

// effectiveLanguage returns the language of the post, falling back to the
// language of the blog it belongs to.
func (p *PublicPost) effectiveLanguage() string {
	if p.Language.Valid && p.Language.String != "" {
		return p.Language.String
	}
	if p.Collection != nil {
		return p.Collection.Language
	}
	return ""
}
