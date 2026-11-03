// Package i18n holds what the server itself says to people: Suki's fixed messages,
// date notes, notifications and email. Text is looked up by its English, as in the
// mobile app, so anything without a translation (a model's reply, a venue's name)
// passes through unchanged.
package i18n

import "strings"

// column maps a language setting to its place in the catalog.
var column = map[string]int{"zh": 0, "ko": 1, "ja": 2, "es": 3}

// T returns english in lang, with {placeholders} filled from key, value pairs.
func T(lang, english string, vars ...string) string {
	text := english
	if c, ok := column[lang]; ok {
		if row, ok := catalog[english]; ok {
			text = row[c]
		}
	}
	for i := 0; i+1 < len(vars); i += 2 {
		text = strings.ReplaceAll(text, "{"+vars[i]+"}", vars[i+1])
	}
	return text
}

// catalog is [Simplified Chinese, Korean, Japanese, Spanish], keyed by the English in the
// code. Suki speaks in the first person, so these keep her voice: warm and casual.
var catalog = map[string][4]string{
	// Suki, in her own conversation.
	"Hi, I'm Suki! I'm an AI matchmaker, and I'm going to help you meet someone you actually click with. No endless swiping, just good introductions. Let's get to know each other. What should I call you?": {
		"嗨，我是 Suki！我是一位 AI 红娘，会帮你遇见真正合拍的人。不用没完没了地滑卡片，只有用心的介绍。我们先认识一下吧，我该怎么称呼你？",
		"안녕하세요, 저는 Suki예요! AI 매치메이커로서 정말 잘 맞는 사람을 만나도록 도와드릴게요. 끝없는 스와이프 없이, 좋은 소개만요. 우리 먼저 서로 알아가요. 뭐라고 불러 드릴까요?",
		"こんにちは、Suki です！AI の仲人として、本当に気の合う人との出会いをお手伝いします。延々とスワイプしなくても大丈夫、いい紹介だけ。まずはお互いを知りましょう。なんて呼べばいいですか？",
		"¡Hola, soy Suki! Soy una casamentera con IA y voy a ayudarte a conocer a alguien con quien de verdad conectes. Nada de deslizar sin fin, solo buenas presentaciones. Vamos a conocernos. ¿Cómo te llamo?"},
	"Hi, I'm Suki! I'm so glad you're here. Let's find you someone wonderful.": {
		"嗨，我是 Suki！很高兴你来了。我们一起去找那个很棒的人吧。",
		"안녕하세요, 저는 Suki예요! 와 주셔서 정말 기뻐요. 멋진 사람을 함께 찾아봐요.",
		"こんにちは、Suki です！来てくれてうれしいです。素敵な人を一緒に見つけましょう。",
		"¡Hola, soy Suki! Me alegra mucho que estés aquí. Vamos a encontrarte a alguien maravilloso."},
	"I found someone! Meet {name}.": {
		"我帮你找到一个人！来认识一下 {name}。",
		"좋은 사람을 찾았어요! {name}님을 소개할게요.",
		"いい人を見つけました！{name} さんを紹介しますね。",
		"¡Encontré a alguien! Te presento a {name}."},
	"It's a match with {name}! You both said yes. I'm thinking of an opener for you right now.": {
		"你和 {name} 配对成功！你们都说好了。我正在帮你想开场白。",
		"{name}님과 매치됐어요! 두 분 모두 좋다고 했어요. 지금 첫마디를 생각하고 있어요.",
		"{name} さんとマッチしました！おふたりとも「はい」でした。今、最初のひと言を考えていますね。",
		"¡Tienes un match con {name}! Los dos dijisteis que sí. Estoy pensando en cómo podrías empezar."},
	"Your photo is approved and looks good to go!": {
		"你的照片通过审核了，看起来很棒！",
		"사진이 승인됐어요. 아주 좋아요!",
		"写真が承認されました。ばっちりです！",
		"¡Tu foto está aprobada y queda genial!"},
	"I can't use that photo. {reason} Want to try another?": {
		"这张照片用不了。{reason}要不要换一张试试？",
		"그 사진은 쓸 수 없어요. {reason} 다른 사진으로 해 볼까요?",
		"その写真は使えません。{reason}別の写真を試してみませんか？",
		"No puedo usar esa foto. {reason} ¿Quieres probar con otra?"},
	"Your profile is ready, and it looks great. I'm off to find people I think you'd really like. I'll be back soon!": {
		"你的资料准备好了，看起来很棒。我这就去找你可能会喜欢的人，很快回来！",
		"프로필이 준비됐어요. 정말 멋져요. 마음에 들 만한 사람들을 찾으러 갈게요. 금방 돌아올게요!",
		"プロフィールが完成しました。とても素敵です。気に入りそうな人を探してきますね。すぐ戻ります！",
		"Tu perfil está listo y queda genial. Voy a buscar personas que creo que te encantarán. ¡Vuelvo pronto!"},
	"No one I'd honestly recommend yet. I'm not going to waste your time on a weak intro. I'll keep looking and tell you as soon as someone good turns up.": {
		"目前还没有我能真心推荐的人。我不想用勉强的介绍浪费你的时间，会继续找，一有合适的人就告诉你。",
		"아직 진심으로 추천할 만한 사람이 없어요. 어설픈 소개로 시간을 뺏고 싶지 않아요. 계속 찾아보고 좋은 사람이 나타나면 바로 알려 드릴게요.",
		"まだ心からおすすめできる人がいません。いまひとつの紹介で時間を無駄にしたくないので、探し続けて、いい人が見つかったらすぐ知らせますね。",
		"Todavía no hay nadie a quien recomendaría de verdad. No voy a hacerte perder el tiempo con una presentación floja. Seguiré buscando y te avisaré en cuanto aparezca alguien bueno."},
	"You've had your introductions for today. Let's see how those go, and I'll be back with more tomorrow.": {
		"今天的介绍已经都给你了。先看看进展如何，明天我再带新的人来。",
		"오늘 소개는 여기까지예요. 어떻게 되는지 지켜보고, 내일 또 소개해 드릴게요.",
		"今日の紹介はここまでです。まずは様子を見てみましょう。また明日、新しい人を連れてきますね。",
		"Ya tienes tus presentaciones de hoy. Veamos cómo van, y mañana vuelvo con más."},
	"My search hit a snag, sorry! I'll try again on my next round, so nothing's lost.": {
		"抱歉，这次搜索出了点小问题！下一轮我会再试一次，什么都不会丢。",
		"죄송해요, 찾는 중에 문제가 생겼어요! 다음 차례에 다시 해 볼게요. 사라진 건 없어요.",
		"ごめんなさい、検索でつまずきました！次の回でもう一度試すので、何も失われていません。",
		"¡Perdona, mi búsqueda tuvo un problema! Lo intentaré de nuevo en la próxima ronda, así que no se pierde nada."},
	"Sorry, my head's a bit foggy right now. I saved what you said. Could you try me again in a little while?": {
		"抱歉，我现在脑子有点转不动。你说的我都存好了，过一会儿再来找我好吗？",
		"미안해요, 지금 머리가 좀 멍해요. 말씀하신 건 저장해 뒀어요. 조금 있다가 다시 말을 걸어 주실래요?",
		"ごめんなさい、今ちょっと頭がぼんやりしています。話してくれたことは保存しました。少ししてからまた声をかけてくれますか？",
		"Perdona, ahora tengo la cabeza un poco nublada. Guardé lo que dijiste. ¿Puedes volver a intentarlo en un rato?"},
	"I've done a lot of thinking today and need a little rest. I'll be fully back tomorrow. Everything you've told me is saved.": {
		"我今天想了太多事，需要休息一下，明天就完全恢复了。你告诉我的都存好了。",
		"오늘 생각을 너무 많이 해서 조금 쉬어야 해요. 내일이면 완전히 돌아올게요. 말씀해 주신 건 모두 저장돼 있어요.",
		"今日はたくさん考えたので少し休みます。明日には完全に戻りますね。話してくれたことはすべて保存してあります。",
		"Hoy he pensado mucho y necesito descansar un poco. Mañana vuelvo del todo. Todo lo que me has contado está guardado."},
	"Your account is paused until someone on our team has taken a look. I can't help with introductions in the meantime. If this seems wrong, write to support@heros-agent.space.": {
		"你的账户已暂停，等我们团队的人看过之后才会恢复。在此期间我没法帮你介绍。如果你觉得有误，请写信到 support@heros-agent.space。",
		"팀원이 확인할 때까지 계정이 일시 중지돼요. 그동안에는 소개를 도와드릴 수 없어요. 잘못된 것 같다면 support@heros-agent.space로 메일을 보내 주세요.",
		"チームの担当者が確認するまで、アカウントは一時停止中です。その間は紹介のお手伝いができません。間違いだと思う場合は support@heros-agent.space までご連絡ください。",
		"Tu cuenta está en pausa hasta que alguien de nuestro equipo la revise. Mientras tanto no puedo ayudarte con presentaciones. Si crees que es un error, escribe a support@heros-agent.space."},
	"Thanks for telling me. I've got it.": {
		"谢谢你告诉我，我记下了。",
		"말해 줘서 고마워요. 알겠어요.",
		"教えてくれてありがとう。わかりました。",
		"Gracias por contármelo. Ya lo tengo."},
	"Go to my matches": {"去看我的配对", "내 매치 보기", "マッチを見る", "Ir a mis matches"},
	"I keep things PG around here, so I'll leave that one alone. Happy to help with your profile, your matches or planning a great date.": {
		"我这里只聊健康的内容，这个话题就不接了。我很乐意帮你弄资料、看配对，或者安排一场很棒的约会。",
		"여기서는 건전한 이야기만 해서 그건 넘어갈게요. 프로필이나 매치, 멋진 데이트 계획은 기꺼이 도와드릴게요.",
		"ここでは健全な話題だけにしているので、それはやめておきますね。プロフィールやマッチ、素敵なデートの計画なら喜んでお手伝いします。",
		"Aquí mantengo las cosas para todos los públicos, así que eso lo dejo pasar. Encantada de ayudarte con tu perfil, tus matches o a planear una gran cita."},

	// Safety replies. These stay plain: no jokes.
	"I'm really glad you told me, and I'm going to drop the jokes for a second. I'm an AI, so I can't be the help you deserve right now, but people can.\n\nIf you might act on this or you're in danger, please call your local emergency number now. In the US or Canada you can call or text 988 any time. Anywhere else, findahelpline.com lists free services for your country.\n\nIf you can, reach out to someone you trust and tell them what you told me. I've let a person on our safety team know.": {
		"你愿意告诉我，我真的很高兴。这一刻我不开玩笑了。我是 AI，给不了你现在应得的帮助，但有人可以。\n\n如果你可能会付诸行动，或者正处于危险中，请马上拨打当地的紧急电话。在美国或加拿大，可以随时拨打或发短信到 988。其他地区可以在 findahelpline.com 查到你所在国家的免费服务。\n\n如果可以，请联系一位你信任的人，把告诉我的话也告诉 TA。我已经通知了我们安全团队的工作人员。",
		"말해 줘서 정말 다행이에요. 잠시 농담은 내려놓을게요. 저는 AI라서 지금 당신에게 필요한 도움을 드릴 수 없지만, 사람은 할 수 있어요.\n\n실행에 옮길 것 같거나 위험한 상황이라면 지금 바로 현지 긴급 전화로 연락해 주세요. 미국이나 캐나다에서는 언제든 988로 전화나 문자를 할 수 있어요. 다른 지역은 findahelpline.com에서 그 나라의 무료 서비스를 찾을 수 있어요.\n\n가능하다면 믿을 수 있는 사람에게 연락해서 제게 한 이야기를 들려주세요. 저희 안전팀 담당자에게도 알렸어요.",
		"話してくれて本当によかったです。少しの間、冗談はやめますね。私は AI なので、今あなたに必要な助けにはなれませんが、人ならなれます。\n\n実行してしまいそうなとき、危険な状況にあるときは、今すぐ現地の緊急番号に電話してください。米国やカナダでは 988 にいつでも電話やテキストができます。その他の地域では findahelpline.com でお住まいの国の無料窓口を探せます。\n\nできれば信頼できる人に連絡して、私に話したことを伝えてください。安全チームの担当者にも知らせました。",
		"Me alegra mucho que me lo hayas contado, y voy a dejar las bromas un momento. Soy una IA, así que no puedo darte la ayuda que mereces ahora mismo, pero las personas sí.\n\nSi crees que podrías hacerlo o estás en peligro, llama ahora al número de emergencias de tu zona. En EE. UU. o Canadá puedes llamar o escribir al 988 a cualquier hora. En otros lugares, findahelpline.com tiene servicios gratuitos de tu país.\n\nSi puedes, habla con alguien de confianza y cuéntale lo que me has contado. Ya avisé a una persona de nuestro equipo de seguridad."},
	"Your safety comes first. If you're in danger right now, call your local emergency number (911 in the US, 112 in the EU) and get to somewhere with other people around.\n\nI've flagged this for a person on our safety team. You can also block this person from their profile, and they'll lose all contact with you straight away.": {
		"你的安全最重要。如果你现在有危险，请拨打当地紧急电话（美国 911，欧盟 112），并尽快去有其他人在的地方。\n\n我已经把这件事转给了我们安全团队的工作人员。你也可以在对方的资料页屏蔽 TA，对方会立刻失去与你的所有联系。",
		"당신의 안전이 가장 중요해요. 지금 위험하다면 현지 긴급 전화(미국 911, EU 112)로 연락하고 주변에 사람이 있는 곳으로 가세요.\n\n저희 안전팀 담당자에게 알렸어요. 상대방의 프로필에서 차단할 수도 있고, 그러면 바로 모든 연락이 끊겨요.",
		"あなたの安全が第一です。今危険な状況なら、現地の緊急番号（米国 911、EU 112）に電話して、周りに人がいる場所へ移動してください。\n\n安全チームの担当者に知らせました。相手のプロフィールからブロックすることもでき、すぐに連絡は一切取れなくなります。",
		"Tu seguridad es lo primero. Si estás en peligro ahora mismo, llama al número de emergencias de tu zona (911 en EE. UU., 112 en la UE) y ve a un lugar con más gente.\n\nYa avisé a una persona de nuestro equipo de seguridad. También puedes bloquear a esta persona desde su perfil, y perderá todo contacto contigo al instante."},
	"Thanks for being honest with me. SynAgent is only for adults aged 18 and over, so I can't help you with dating here, and I've paused this account until someone on our team has looked at it. No hard feelings.": {
		"谢谢你对我说实话。SynAgent 只面向 18 岁及以上的成年人，所以我没法在这里帮你约会，这个账户也已暂停，等我们团队的人看过再说。别往心里去。",
		"솔직하게 말해 줘서 고마워요. SynAgent는 만 18세 이상 성인만 이용할 수 있어서 여기서는 데이트를 도와드릴 수 없고, 팀원이 확인할 때까지 이 계정을 일시 중지했어요. 기분 나쁘게 생각하지 말아요.",
		"正直に話してくれてありがとう。SynAgent は 18 歳以上の成人専用なので、ここでデートのお手伝いはできません。チームの担当者が確認するまで、このアカウントは一時停止しました。悪く思わないでくださいね。",
		"Gracias por ser sincero conmigo. SynAgent es solo para adultos de 18 años o más, así que no puedo ayudarte con citas aquí, y he pausado esta cuenta hasta que alguien de nuestro equipo la revise. Sin rencores."},
	"Feedback after dates privately shapes who I introduce you to, and those details stay private, so there's no number for you to look at. The privacy policy explains more. So tell me: what made your favorite date your favorite?": {
		"约会后的反馈会在私下影响我之后给你介绍谁，这些细节是保密的，所以没有数字可以给你看。隐私政策里有更多说明。那告诉我：你最喜欢的一次约会，好在哪里？",
		"데이트 후 피드백은 제가 다음에 누구를 소개할지에 비공개로 영향을 줘요. 자세한 내용은 비공개라서 보여 드릴 숫자는 없어요. 더 알고 싶다면 개인정보 처리방침을 참고해 주세요. 그럼 말해 주세요. 가장 좋았던 데이트는 뭐가 좋았나요?",
		"デート後のフィードバックは、次に誰を紹介するかに非公開で影響します。詳細は非公開なので、お見せできる数字はありません。詳しくはプライバシーポリシーをご覧ください。では教えてください。一番よかったデートは、どこがよかったですか？",
		"Los comentarios después de las citas influyen en privado en a quién te presento, y esos detalles son privados, así que no hay ningún número que puedas ver. La política de privacidad explica más. Cuéntame: ¿qué hizo que tu cita favorita fuera tu favorita?"},
	"Feedback after dates privately shapes who I introduce you to, and those details stay private. The privacy policy explains more. Tell me more about what you're looking for, and I'll keep it in mind for your next introduction.": {
		"约会后的反馈会在私下影响我之后给你介绍谁，这些细节是保密的。隐私政策里有更多说明。再多跟我说说你想找什么样的人，下次介绍时我会记在心上。",
		"데이트 후 피드백은 제가 다음에 누구를 소개할지에 비공개로 영향을 주고, 자세한 내용은 비공개예요. 더 알고 싶다면 개인정보 처리방침을 참고해 주세요. 어떤 사람을 찾는지 더 들려주면 다음 소개 때 기억해 둘게요.",
		"デート後のフィードバックは、次に誰を紹介するかに非公開で影響し、詳細は非公開です。詳しくはプライバシーポリシーをご覧ください。どんな人を探しているかもっと教えてくれたら、次の紹介で覚えておきますね。",
		"Los comentarios después de las citas influyen en privado en a quién te presento, y esos detalles son privados. La política de privacidad explica más. Cuéntame más sobre lo que buscas y lo tendré en cuenta para tu próxima presentación."},
	"I'm sorry, that's not okay and it isn't on you. I've flagged it for a person on our safety team. If you want them gone from your life right now, blocking removes all contact straight away. Want to do that?": {
		"我很抱歉，这样很不对，也不是你的错。我已经转给了我们安全团队的工作人员。如果你想让 TA 马上从你的生活里消失，屏蔽会立刻切断所有联系。要这样做吗？",
		"정말 미안해요. 그건 잘못된 일이고 당신 탓이 아니에요. 저희 안전팀 담당자에게 알렸어요. 지금 바로 그 사람과 끊고 싶다면 차단하면 모든 연락이 즉시 끊겨요. 그렇게 할까요?",
		"つらかったですね。それは許されないことで、あなたのせいではありません。安全チームの担当者に知らせました。今すぐ関わりを断ちたければ、ブロックすると連絡はすぐに一切取れなくなります。そうしますか？",
		"Lo siento, eso no está bien y no es culpa tuya. Ya avisé a una persona de nuestro equipo de seguridad. Si quieres que desaparezca de tu vida ahora mismo, al bloquearla se corta todo contacto al instante. ¿Quieres hacerlo?"},
	"Please don't send money, gift cards or crypto to anyone you've met here. It's the classic sign of a scam, however good the story sounds, and real matches don't ask. I've flagged it for our safety team. Do you want to block them?": {
		"请不要给在这里认识的任何人转钱、送礼品卡或加密货币。不管故事听起来多动人，这都是典型的诈骗信号，真心的人不会开这个口。我已经转给了安全团队。要屏蔽 TA 吗？",
		"여기서 만난 사람에게 돈, 기프트 카드, 암호화폐를 보내지 마세요. 아무리 그럴듯한 이야기여도 전형적인 사기 신호이고, 진짜 매치는 그런 걸 요구하지 않아요. 안전팀에 알렸어요. 차단할까요?",
		"ここで出会った人に、お金やギフトカード、暗号資産を送らないでください。どんなにもっともらしい話でも典型的な詐欺のサインで、本当のマッチはそんなことを頼みません。安全チームに知らせました。ブロックしますか？",
		"Por favor, no envíes dinero, tarjetas regalo ni criptomonedas a nadie que hayas conocido aquí. Es la señal clásica de una estafa, por muy buena que suene la historia, y los matches de verdad no lo piden. Ya avisé a nuestro equipo de seguridad. ¿Quieres bloquearla?"},

	// Photo review.
	"This photo isn't allowed.":                                                  {"这张照片不被允许。", "이 사진은 허용되지 않아요.", "この写真は使用できません。", "Esta foto no está permitida."},
	"A person on our team will take a look at this one.":                         {"我们团队的人会看一下这张。", "저희 팀원이 이 사진을 확인할 거예요.", "チームの担当者がこの写真を確認します。", "Una persona de nuestro equipo revisará esta."},
	"Nudity and sexual content aren't allowed.":                                  {"不允许裸露和色情内容。", "노출과 성적인 콘텐츠는 허용되지 않아요.", "裸や性的な内容は禁止です。", "No se permiten desnudos ni contenido sexual."},
	"This photo shows something that isn't allowed here.":                        {"这张照片里有这里不允许的内容。", "이 사진에는 여기서 허용되지 않는 내용이 있어요.", "この写真にはここで許可されていないものが写っています。", "Esta foto muestra algo que no se permite aquí."},
	"Please use a photo without contact details in it.":                          {"请使用不含联系方式的照片。", "연락처가 없는 사진을 사용해 주세요.", "連絡先が写っていない写真を使ってください。", "Usa una foto que no incluya datos de contacto."},
	"I need a photo that clearly shows your face.":                               {"我需要一张能看清你脸的照片。", "얼굴이 잘 보이는 사진이 필요해요.", "顔がはっきり写った写真が必要です。", "Necesito una foto en la que se vea bien tu cara."},
	"For your main photos, please use one where it's clear which person is you.": {"主照片请选一张能看出哪位是你的。", "대표 사진은 누가 본인인지 분명한 것으로 해 주세요.", "メイン写真には、どの人があなたかわかるものを使ってください。", "Para tus fotos principales, usa una en la que quede claro quién eres."},

	// Date notes. One message, read by two people, so these are rendered for whoever is reading.
	"Suki is looking for a place you'd both enjoy.": {"Suki 正在找你们都会喜欢的地方。", "Suki가 두 분 모두 좋아할 장소를 찾고 있어요.", "Suki がおふたりとも楽しめる場所を探しています。", "Suki está buscando un sitio que os guste a los dos."},
	"I found a few places you'd both like. Pick your favorite, and your pick stays private until you've both chosen.": {
		"我找到了几个你们都会喜欢的地方。选一个最喜欢的吧，等你们都选好之前，选择是保密的。",
		"두 분 모두 좋아할 만한 장소를 몇 곳 찾았어요. 마음에 드는 곳을 고르세요. 둘 다 고를 때까지 선택은 비공개예요.",
		"おふたりが気に入りそうな場所をいくつか見つけました。お気に入りを選んでください。ふたりとも選ぶまで、選択は非公開です。",
		"Encontré algunos sitios que os gustarán a los dos. Elige tu favorito; tu elección es privada hasta que ambos hayáis elegido."},
	"I couldn't find good places nearby just now. {why} You can ask me to try again, or pick somewhere yourselves.": {
		"我暂时没在附近找到合适的地方。{why}你们可以让我再试一次，或者自己选个地方。",
		"지금은 근처에서 좋은 장소를 찾지 못했어요. {why} 다시 찾아 달라고 하거나 직접 골라 보세요.",
		"今は近くにいい場所が見つかりませんでした。{why}もう一度頼むか、おふたりで選んでください。",
		"Ahora mismo no encontré buenos sitios cerca. {why} Podéis pedirme que lo intente de nuevo o elegir uno vosotros."},
	"The map doesn't list enough places near you yet.":                             {"地图上你们附近的地点还不够多。", "지도에 근처 장소가 아직 충분하지 않아요.", "地図にはまだ近くの場所が十分に載っていません。", "El mapa aún no tiene suficientes sitios cerca de vosotros."},
	"The search didn't work this time.":                                            {"这次搜索没有成功。", "이번 검색은 잘되지 않았어요.", "今回の検索はうまくいきませんでした。", "Esta vez la búsqueda no funcionó."},
	"It's {venue}! Now, when works for you both?":                                  {"就去 {venue}！那么，你们什么时候方便？", "{venue}(으)로 정해졌어요! 이제 언제가 좋을까요?", "{venue} に決まりました！では、いつがいいですか？", "¡Será en {venue}! Ahora, ¿cuándo os viene bien a los dos?"},
	"New time suggested. Does it work?":                                            {"有人提议了新时间，可以吗？", "새로운 시간이 제안됐어요. 괜찮으세요?", "新しい時間が提案されました。大丈夫ですか？", "Se ha propuesto una nueva hora. ¿Os va bien?"},
	"You're on! Have a wonderful time. Remember to tell a friend where you'll be.": {"定了！祝你们玩得开心。记得告诉朋友你们会去哪里。", "확정이에요! 즐거운 시간 보내세요. 어디 가는지 친구에게 꼭 알려 주세요.", "決まり！楽しんできてくださいね。行き先を友だちに伝えるのを忘れずに。", "¡Hecho! Que lo paséis genial. Recordad contarle a un amigo dónde estaréis."},
	"This date plan was cancelled.":                                                {"这个约会计划已取消。", "이 데이트 계획은 취소됐어요.", "このデートの計画はキャンセルされました。", "Este plan de cita se ha cancelado."},
	"I'm meeting {name} at {venue}{address} on {time}. Just letting you know where I'll be.": {
		"我会在 {time} 去 {venue}{address} 见 {name}。告诉你一声我会在哪儿。",
		"{time}에 {venue}{address}에서 {name}님을 만나요. 제가 어디 있을지 알려 드려요.",
		"{time} に {venue}{address} で {name} さんと会います。居場所をお知らせしておきますね。",
		"Voy a quedar con {name} en {venue}{address} el {time}. Solo para que sepas dónde estaré."},

	// Notifications.
	"Suki found someone for you":                    {"Suki 为你找到了一个人", "Suki가 좋은 사람을 찾았어요", "Suki がいい人を見つけました", "Suki ha encontrado a alguien para ti"},
	"Take a look when you have a moment.":           {"有空时看一眼吧。", "시간 날 때 한번 보세요.", "時間のあるときに見てみてください。", "Échale un vistazo cuando puedas."},
	"It's a match!":                                 {"配对成功！", "매치됐어요!", "マッチしました！", "¡Tienes un match!"},
	"You and {name} both said yes.":                 {"你和 {name} 都说好了。", "{name}님과 두 분 모두 좋다고 했어요.", "{name} さんとおふたりとも「はい」でした。", "Tú y {name} habéis dicho que sí."},
	"New message":                                   {"新消息", "새 메시지", "新しいメッセージ", "Mensaje nuevo"},
	"You have a new message.":                       {"你有一条新消息。", "새 메시지가 왔어요.", "新しいメッセージがあります。", "Tienes un mensaje nuevo."},
	"Date ideas are ready":                          {"约会地点选好了", "데이트 장소가 준비됐어요", "デートのアイデアができました", "Las ideas para la cita están listas"},
	"Suki found a few places for you both.":         {"Suki 为你们找到了几个地方。", "Suki가 두 분을 위한 장소를 몇 곳 찾았어요.", "Suki がおふたりに合う場所をいくつか見つけました。", "Suki ha encontrado algunos sitios para los dos."},
	"A time for your date":                          {"约会时间提议", "데이트 시간 제안", "デートの時間", "Una hora para vuestra cita"},
	"Open SynAgent to say yes or suggest another.":  {"打开 SynAgent 确认，或者提议别的时间。", "SynAgent를 열어 수락하거나 다른 시간을 제안하세요.", "SynAgent を開いて、承諾するか別の時間を提案してください。", "Abre SynAgent para aceptar o proponer otra."},
	"Your date is confirmed":                        {"约会已确认", "데이트가 확정됐어요", "デートが確定しました", "Tu cita está confirmada"},
	"See you there. Suki will check in afterwards.": {"祝顺利。之后 Suki 会来问问你。", "잘 다녀오세요. 나중에 Suki가 안부를 물을게요.", "楽しんできてください。あとで Suki が様子を聞きますね。", "Que vaya bien. Suki te preguntará después."},
	"Your date is tomorrow":                         {"明天就是约会了", "내일이 데이트예요", "デートは明日です", "Tu cita es mañana"},
	"Tell a friend your plans, and have fun.":       {"把安排告诉朋友，玩得开心。", "친구에게 계획을 알리고 즐거운 시간 보내세요.", "予定を友だちに伝えて、楽しんでください。", "Cuéntale tus planes a un amigo y pásalo bien."},
	"Your date is coming up":                        {"约会快到了", "데이트가 곧이에요", "もうすぐデートです", "Tu cita está cerca"},
	"It's almost time.":                             {"时间快到了。", "거의 시간이 됐어요.", "もうすぐ時間です。", "Ya casi es la hora."},
	"Everything okay?":                              {"一切还好吗？", "괜찮으세요?", "大丈夫ですか？", "¿Todo bien?"},
	"Just checking in. If anything feels off, open SynAgent and tell Suki.": {"来问候一下。如果有什么不对劲，打开 SynAgent 告诉 Suki。", "잘 지내는지 확인해요. 뭔가 이상하다면 SynAgent를 열어 Suki에게 말해 주세요.", "様子を聞きに来ました。何か変だと感じたら、SynAgent を開いて Suki に伝えてください。", "Solo quería saber cómo estás. Si algo no va bien, abre SynAgent y cuéntaselo a Suki."},
	"How did it go?":                    {"约会怎么样？", "어땠어요?", "どうでしたか？", "¿Qué tal fue?"},
	"Suki would love to hear about it.": {"Suki 很想听听。", "Suki가 듣고 싶어 해요.", "Suki が聞きたがっています。", "A Suki le encantaría saberlo."},

	// Email.
	"Open SynAgent":            {"打开 SynAgent", "SynAgent 열기", "SynAgent を開く", "Abrir SynAgent"},
	"Open SynAgent to see it.": {"打开 SynAgent 查看。", "SynAgent를 열어 확인하세요.", "SynAgent を開いて確認してください。", "Abre SynAgent para verlo."},
	"You can turn these emails off in Settings > Notifications.":                               {"你可以在“设置 > 通知”中关闭这些邮件。", "설정 > 알림에서 이 이메일을 끌 수 있어요.", "これらのメールは「設定 > 通知」でオフにできます。", "Puedes desactivar estos correos en Ajustes > Notificaciones."},
	"Your SynAgent verification code":                                                          {"你的 SynAgent 验证码", "SynAgent 인증 코드", "SynAgent の確認コード", "Tu código de verificación de SynAgent"},
	"Welcome! Here is the code to confirm your email address.":                                 {"欢迎！这是确认邮箱地址的验证码。", "환영해요! 이메일 주소를 확인할 코드예요.", "ようこそ！メールアドレスを確認するためのコードです。", "¡Bienvenido! Aquí tienes el código para confirmar tu correo."},
	"If you didn't sign up, you can ignore this email.":                                        {"如果不是你注册的，忽略这封邮件即可。", "가입한 적이 없다면 이 이메일은 무시하세요.", "登録した覚えがなければ、このメールは無視してください。", "Si no te registraste, puedes ignorar este correo."},
	"Your SynAgent password reset code":                                                        {"你的 SynAgent 密码重置验证码", "SynAgent 비밀번호 재설정 코드", "SynAgent のパスワード再設定コード", "Tu código para restablecer la contraseña de SynAgent"},
	"Here is the code to choose a new password.":                                               {"这是设置新密码的验证码。", "새 비밀번호를 정할 코드예요.", "新しいパスワードを設定するためのコードです。", "Aquí tienes el código para elegir una contraseña nueva."},
	"If you didn't ask for this, your password has not changed and you can ignore this email.": {"如果不是你申请的，你的密码没有变化，忽略这封邮件即可。", "요청한 적이 없다면 비밀번호는 바뀌지 않았으니 이 이메일은 무시하세요.", "心当たりがなければ、パスワードは変更されていないので、このメールは無視してください。", "Si no lo pediste, tu contraseña no ha cambiado y puedes ignorar este correo."},
	"Hi":                       {"你好", "안녕하세요", "こんにちは", "Hola"},
	"Hi {name}":                {"{name}，你好", "{name}님, 안녕하세요", "{name} さん、こんにちは", "Hola, {name}"},
	"It works for 15 minutes.": {"15 分钟内有效。", "15분 동안 유효해요.", "15 分間有効です。", "Funciona durante 15 minutos."},
}
