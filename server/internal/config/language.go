package config

type Language = string

const (
	LangEnglish    Language = "en"
	LangIndonesian Language = "id"
	LangSpanish    Language = "es"
	LangFrench     Language = "fr"
	LangGerman     Language = "de"
	LangPortuguese Language = "pt"
	LangItalian    Language = "it"
	LangJapanese   Language = "ja"
	LangKorean     Language = "ko"
	LangChinese    Language = "zh"
	LangArabic     Language = "ar"
	LangHindi      Language = "hi"
	LangRussian    Language = "ru"
	LangDutch      Language = "nl"
	LangTurkish    Language = "tr"
)

var CodeToLanguage = map[string]Language{
	"en-US": LangEnglish,
	"id-ID": LangIndonesian,
	"es-ES": LangSpanish,
	"fr-FR": LangFrench,
	"de-DE": LangGerman,
	"pt-BR": LangPortuguese,
	"it-IT": LangItalian,
	"ja-JP": LangJapanese,
	"ko-KR": LangKorean,
	"zh-CN": LangChinese,
	"ar-SA": LangArabic,
	"hi-IN": LangHindi,
	"ru-RU": LangRussian,
	"nl-NL": LangDutch,
	"tr-TR": LangTurkish,
}

var LanguageToCode = map[Language]string{
	LangEnglish:    "en-US",
	LangIndonesian: "id-ID",
	LangSpanish:    "es-ES",
	LangFrench:     "fr-FR",
	LangGerman:     "de-DE",
	LangPortuguese: "pt-BR",
	LangItalian:    "it-IT",
	LangJapanese:   "ja-JP",
	LangKorean:     "ko-KR",
	LangChinese:    "zh-CN",
	LangArabic:     "ar-SA",
	LangHindi:      "hi-IN",
	LangRussian:    "ru-RU",
	LangDutch:      "nl-NL",
	LangTurkish:    "tr-TR",
}

var LanguageNames = map[string]string{
	"en-US": "English",
	"id-ID": "Bahasa Indonesia",
	"es-ES": "Spanish",
	"fr-FR": "French",
	"de-DE": "German",
	"pt-BR": "Portuguese",
	"it-IT": "Italian",
	"ja-JP": "Japanese",
	"ko-KR": "Korean",
	"zh-CN": "Chinese (Mandarin)",
	"ar-SA": "Arabic",
	"hi-IN": "Hindi",
	"ru-RU": "Russian",
	"nl-NL": "Dutch",
	"tr-TR": "Turkish",
}

func GetLanguage(code string) Language {
	if lang, ok := CodeToLanguage[code]; ok {
		return lang
	}
	return LangEnglish
}

func GetCode(language Language) string {
	if code, ok := LanguageToCode[language]; ok {
		return code
	}
	return "en-US"
}

func GetLanguageName(code string) string {
	if name, ok := LanguageNames[code]; ok {
		return name
	}
	return "English"
}
