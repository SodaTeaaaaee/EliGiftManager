const KATAKANA_TO_HIRAGANA: Record<string, string> = {
  "ア": "あ",
  "イ": "い",
  "ウ": "う",
  "エ": "え",
  "オ": "お",
  "ァ": "あ",
  "ィ": "い",
  "ゥ": "う",
  "ェ": "え",
  "ォ": "お",
  "カ": "か",
  "キ": "き",
  "ク": "く",
  "ケ": "け",
  "コ": "こ",
  "サ": "さ",
  "シ": "し",
  "ス": "す",
  "セ": "せ",
  "ソ": "そ",
  "タ": "た",
  "チ": "ち",
  "ツ": "つ",
  "テ": "て",
  "ト": "と",
  "ナ": "な",
  "ニ": "に",
  "ヌ": "ぬ",
  "ネ": "ね",
  "ノ": "の",
  "ハ": "は",
  "ヒ": "ひ",
  "フ": "ふ",
  "ヘ": "へ",
  "ホ": "ほ",
  "マ": "ま",
  "ミ": "み",
  "ム": "む",
  "メ": "め",
  "モ": "も",
  "ヤ": "や",
  "ユ": "ゆ",
  "ヨ": "よ",
  "ラ": "ら",
  "リ": "り",
  "ル": "る",
  "レ": "れ",
  "ロ": "ろ",
  "ワ": "わ",
  "ヲ": "を",
  "ン": "ん",
  "ガ": "が",
  "ギ": "ぎ",
  "グ": "ぐ",
  "ゲ": "げ",
  "ゴ": "ご",
  "ザ": "ざ",
  "ジ": "じ",
  "ズ": "ず",
  "ゼ": "ぜ",
  "ゾ": "ぞ",
  "ダ": "だ",
  "ヂ": "ぢ",
  "ヅ": "づ",
  "デ": "で",
  "ド": "ど",
  "バ": "ば",
  "ビ": "び",
  "ブ": "ぶ",
  "ベ": "べ",
  "ボ": "ぼ",
  "パ": "ぱ",
  "ピ": "ぴ",
  "プ": "ぷ",
  "ペ": "ぺ",
  "ポ": "ぽ",
  "ャ": "ゃ",
  "ュ": "ゅ",
  "ョ": "ょ",
  "ッ": "っ",
  "ー": "ー",
};

const HIRAGANA_ROMAJI: Record<string, string> = {
  "あ": "a",
  "い": "i",
  "う": "u",
  "え": "e",
  "お": "o",
  "か": "ka",
  "き": "ki",
  "く": "ku",
  "け": "ke",
  "こ": "ko",
  "さ": "sa",
  "し": "shi",
  "す": "su",
  "せ": "se",
  "そ": "so",
  "た": "ta",
  "ち": "chi",
  "つ": "tsu",
  "て": "te",
  "と": "to",
  "な": "na",
  "に": "ni",
  "ぬ": "nu",
  "ね": "ne",
  "の": "no",
  "は": "ha",
  "ひ": "hi",
  "ふ": "fu",
  "へ": "he",
  "ほ": "ho",
  "ま": "ma",
  "み": "mi",
  "む": "mu",
  "め": "me",
  "も": "mo",
  "や": "ya",
  "ゆ": "yu",
  "よ": "yo",
  "ら": "ra",
  "り": "ri",
  "る": "ru",
  "れ": "re",
  "ろ": "ro",
  "わ": "wa",
  "を": "wo",
  "ん": "n",
  "が": "ga",
  "ぎ": "gi",
  "ぐ": "gu",
  "げ": "ge",
  "ご": "go",
  "ざ": "za",
  "じ": "ji",
  "ず": "zu",
  "ぜ": "ze",
  "ぞ": "zo",
  "だ": "da",
  "ぢ": "ji",
  "づ": "zu",
  "で": "de",
  "ど": "do",
  "ば": "ba",
  "び": "bi",
  "ぶ": "bu",
  "べ": "be",
  "ぼ": "bo",
  "ぱ": "pa",
  "ぴ": "pi",
  "ぷ": "pu",
  "ぺ": "pe",
  "ぽ": "po",
  "きゃ": "kya",
  "きゅ": "kyu",
  "きょ": "kyo",
  "しゃ": "sha",
  "しゅ": "shu",
  "しょ": "sho",
  "ちゃ": "cha",
  "ちゅ": "chu",
  "ちょ": "cho",
  "にゃ": "nya",
  "にゅ": "nyu",
  "にょ": "nyo",
  "ひゃ": "hya",
  "ひゅ": "hyu",
  "ひょ": "hyo",
  "みゃ": "mya",
  "みゅ": "myu",
  "みょ": "myo",
  "りゃ": "rya",
  "りゅ": "ryu",
  "りょ": "ryo",
  "ぎゃ": "gya",
  "ぎゅ": "gyu",
  "ぎょ": "gyo",
  "じゃ": "ja",
  "じゅ": "ju",
  "じょ": "jo",
  "びゃ": "bya",
  "びゅ": "byu",
  "びょ": "byo",
  "ぴゃ": "pya",
  "ぴゅ": "pyu",
  "ぴょ": "pyo",
  "っ": "", // sokuon — handled specially below
  "ー": "", // long vowel — handled specially below
};

export function toHiragana(input: string): string {
  let result = "";
  for (let i = 0; i < input.length; i++) {
    // Check for two-char kana (拗音 small ya/yu/yo)
    if (i + 1 < input.length) {
      const two = input.substring(i, i + 2);
      if (KATAKANA_TO_HIRAGANA[two]) {
        result += KATAKANA_TO_HIRAGANA[two];
        i++;
        continue;
      }
    }
    result += KATAKANA_TO_HIRAGANA[input[i]] ?? input[i];
  }
  return result;
}

export function buildKanaRomajiKey(input: string): string {
  const hira = toHiragana(input);
  let result = "";
  let i = 0;
  while (i < hira.length) {
    // Check for two-char kana
    if (i + 1 < hira.length) {
      const two = hira.substring(i, i + 2);
      if (HIRAGANA_ROMAJI[two]) {
        result += HIRAGANA_ROMAJI[two];
        i += 2;
        continue;
      }
    }
    const ch = hira[i];
    if (ch === "っ" || ch === "ッ") {
      // Sokuon: double the next consonant
      i++;
      if (i < hira.length) {
        const next = HIRAGANA_ROMAJI[hira[i]] ?? "";
        result += next.charAt(0) + next;
      }
      continue;
    }
    if (ch === "ー") {
      // Long vowel: repeat the last character
      if (result.length > 0) result += result.charAt(result.length - 1);
      i++;
      continue;
    }
    result += HIRAGANA_ROMAJI[ch] ?? ch;
    i++;
  }
  return result;
}

const HIRAGANA_START = 0x3040;
const HIRAGANA_END = 0x309f;
const KATAKANA_START = 0x30a0;
const KATAKANA_END = 0x30ff;

export function hasKana(input: string): boolean {
  for (const ch of input) {
    const cp = ch.codePointAt(0)!;
    if (
      (cp >= HIRAGANA_START && cp <= HIRAGANA_END) ||
      (cp >= KATAKANA_START && cp <= KATAKANA_END)
    ) {
      return true;
    }
  }
  return false;
}

const HAN_REGEX = /\p{Script=Han}/u;

export function hasHan(input: string): boolean {
  return HAN_REGEX.test(input);
}

const HANGUL_REGEX = /\p{Script=Hangul}/u;

export function hasHangul(input: string): boolean {
  return HANGUL_REGEX.test(input);
}

// Returns true if the string consists ONLY of kana (no kanji, no hangul)
export function isPureKana(input: string): boolean {
  if (input.length === 0) return false;
  for (const ch of input) {
    const cp = ch.codePointAt(0)!;
    if (cp >= HIRAGANA_START && cp <= HIRAGANA_END) continue;
    if (cp >= KATAKANA_START && cp <= KATAKANA_END) continue;
    if (ch === "ー") continue;
    return false;
  }
  return true;
}
