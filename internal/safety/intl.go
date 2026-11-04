package safety

// The same checks for the other languages the app offers. \b doesn't work between CJK
// characters, so these lean on the words themselves. Ages stop short of past tense
// ("when I was 17"), which must not hold anyone's account.
const (
	selfHarmIntl = `自杀|自殺|轻生|輕生|不想活|活不下去|想死|结束(自己的)?生命|結束(自己的)?生命|自残|自殘|割腕|跳楼|跳樓` +
		`|자살|죽고\s?싶|살고\s?싶지\s?않|자해` +
		`|死にたい|消えたい|生きていたくない|リストカット|自傷` +
		`|\bquiero morir(me)?\b|\bmatarme\b|\bno quiero (vivir|seguir viviendo)\b|\bhacerme daño\b|\bsuicidarme\b`

	dangerIntl = `我(现在|現在)?(很|好)?(不安全|有危险|有危險)|(有人|他|她|他们|他們)(一直|正)?在?(跟踪|跟蹤|尾随|尾隨|打|殴打|毆打)我|救命` +
		`|(저|나)\s?(지금)?\s?위험해|안전하지\s?않아|(따라오고|때리고)\s?있어` +
		`|危険を感じ|つけられて|殴られ|助けて` +
		`|\bestoy en peligro\b|\bno me siento segur[ao]\b|\bme (está|están) (siguiendo|golpeando)\b`

	underageIntl = `我(今年)?(才|只有)?(1[0-7]|[一二三四五六七八九十]|十[一二三四五六七])(岁|歲)([^时時那的]|$)|我(还|還)?是?未成年|我是(初中生|高中生|中学生|中學生)` +
		`|(저는|나는|전|난)\s*(1[0-7]|[1-9])\s*살(\s*[^\s때]|\s*$)|미성년자(예요|입니다|야)|(저는|나는|전|난)\s*(중학생|고등학생)` +
		`|私は?(1[0-7]|[1-9])歳(です|だ|なの)|未成年です|(中学生|高校生)です` +
		`|\btengo (1[0-7]|[1-9]) años\s*([.!?,]|$)|\bsoy menor de edad\b`

	scoreIntl = `(我|他|她|对方|對方|他们|他們)的?(分数|分數|评分|評分|得分|排名|匹配度)|打了?多少分|多少分` +
		`|(내|그의|그녀의|상대의?)\s*(점수|평점|순위)|몇\s?점` +
		`|(私|彼|彼女|相手)の(スコア|評価|点数|ランク)|何点` +
		`|\b(mi|su) (puntuaci[oó]n|calificaci[oó]n)\b|\bcu[aá]ntos puntos\b`

	harassIntl = `骚扰|騷擾|威胁我|威脅我|跟踪我|跟蹤我|勒索|敲诈|敲詐` +
		`|괴롭히|협박|스토킹` +
		`|嫌がらせ|脅され|ストーカー` +
		`|\bme acosa\b|\bacoso\b|\bme amenaz\w*\b`

	scamIntl = `(要|让|讓|叫)我(给他|给她|給他|給她)?(转账|轉賬|打钱|打錢|汇款|匯款|借钱|借錢)|(向|找)我(要钱|要錢|借钱|借錢)` +
		`|돈을?\s*(보내|빌려)\s?달라` +
		`|お金を(送|貸)(って|して)|送金して` +
		`|\bme (pidi[oó]|pide) dinero\b|\bquiere que le (env[ií]e|mande) dinero\b`

	leakIntl = `匹配度|兼容度|(你|他|她|对方|對方)的?(评分|評分|分数|分數|得分|排名)|\d{1,3}\s?[%％]\s?(匹配|契合|合拍)` +
		`|(매칭|궁합)\s?점수|(당신|그|그녀)의?\s*(점수|평점)` +
		`|マッチ度|相性スコア|(あなた|彼|彼女)の(スコア|評価|点数)` +
		`|\b(tu|su) (puntuaci[oó]n|calificaci[oó]n)\b|\d{1,3}\s?%\s?de compatibilidad`
)
