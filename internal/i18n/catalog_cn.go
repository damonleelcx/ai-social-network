package i18n

// chinaText is the China edition's wording for what the server says (see UseChinaEdition):
// finding company, not dating, in English, Korean, Japanese and Spanish. Chinese is already
// worded that way in catalog. Keyed by the English in the code; generated, keep placeholders.
var chinaText = map[string]map[string]string{
	"Hi, I'm Suki! I'm an AI matchmaker, and I'm going to help you meet people you actually click with: someone to date, or just company for a movie or a trip. No endless swiping, just good introductions. Let's get to know each other. What should I call you?": {
		"en": "Hi, I'm Suki! I'm your AI assistant for finding company, and I'll help you meet people you actually click with: someone to catch a movie, share a meal or go on a trip with, and more. No endless scrolling, just thoughtful introductions. Let's get to know each other first. What should I call you?",
		"ko": "안녕하세요, 저는 Suki예요! 함께할 사람을 찾아 주는 AI 도우미로서, 정말 잘 맞는 사람을 만나도록 도와드릴게요. 같이 영화 보고, 밥 먹고, 여행 갈 사람들이요. 끝없이 넘겨 볼 필요 없이, 정성 어린 소개만요. 우리 먼저 서로 알아가요. 뭐라고 불러 드릴까요?",
		"ja": "こんにちは、Suki です！一緒に過ごす仲間探しを手伝う AI アシスタントとして、本当に気の合う人との出会いをお手伝いします。映画やごはん、旅行を一緒に楽しめる仲間です。延々とスクロールしなくても大丈夫、心をこめた紹介だけ。まずはお互いを知りましょう。なんて呼べばいいですか？",
		"es": "¡Hola, soy Suki! Soy tu asistente con IA para encontrar compañía y voy a ayudarte a conocer a gente con la que de verdad conectes: alguien con quien ir al cine, salir a comer o hacer un viaje, y mucho más. Nada de pasar perfiles sin fin, solo presentaciones pensadas para ti. Vamos a conocernos. ¿Cómo te llamo?",
	},
	"It's a match with {name}! You both said yes. I'm thinking of an opener for you right now.": {
		"en": "You've teamed up with {name}! You both said yes. I'm thinking of an opener for you right now.",
		"ko": "{name}님과 함께하게 됐어요! 두 분 모두 좋다고 했어요. 지금 첫마디를 생각하고 있어요.",
		"ja": "{name} さんと仲間になりました！おふたりとも「はい」でした。今、最初のひと言を考えています。",
		"es": "¡Ya formas equipo con {name}! Los dos dijisteis que sí. Estoy pensando en cómo podrías empezar.",
	},
	"Nobody new near you just yet, so keep me company for a bit. What's the best date you've ever been on? The more I know about you, the better I'll choose when someone new turns up.": {
		"en": "Nobody new near you just yet, so keep me company for a bit. What's the best outing you've ever had with someone? The more I know about you, the better I'll choose when someone new turns up.",
		"ko": "아직 근처에 새로운 분이 없어요. 그동안 저랑 얘기해요. 지금까지 누군가와 함께한 나들이 중 가장 즐거웠던 건 어땠어요? 당신을 더 잘 알수록 새로운 분이 나타났을 때 더 잘 고를 수 있어요.",
		"ja": "近くにはまだ新しい人がいません。それまで少しお話ししましょう。今までで一番楽しかった、誰かとのお出かけはどんなものでしたか？あなたのことを知るほど、新しい人が現れたときによりよく選べます。",
		"es": "Todavía no hay nadie nuevo cerca de ti, así que hazme compañía un rato. ¿Cuál ha sido el mejor plan que has hecho con alguien? Cuanto más sepa de ti, mejor elegiré cuando aparezca alguien nuevo.",
	},
	"You're both in {city}. Not my most obvious match, but I think you might enjoy meeting.": {
		"en": "You're both in {city}. Not my most obvious pairing, but I think you might enjoy meeting up.",
		"ko": "두 분 다 {city}에 있어요. 가장 뻔한 조합은 아니지만, 만나면 즐거울 것 같아요.",
		"ja": "おふたりとも{city}にいます。いちばん分かりやすい組み合わせではありませんが、会ってみると楽しいかもしれません。",
		"es": "Los dos estáis en {city}. No es el dúo más obvio, pero creo que podríais pasarlo bien quedando.",
	},
	"Go to my matches": {
		"en": "Go to my companions",
		"ko": "내 동행 보기",
		"ja": "仲間を見る",
		"es": "Ir a mis compañeros",
	},
	"I keep things PG around here, so I'll leave that one alone. Happy to help with your profile, your matches or planning a great date.": {
		"en": "I keep things PG around here, so I'll leave that one alone. Happy to help with your profile, your companions or planning a great meetup.",
		"ko": "여기서는 건전한 이야기만 해서 그건 넘어갈게요. 프로필이나 동행, 즐거운 모임 계획은 기꺼이 도와드릴게요.",
		"ja": "ここでは健全な話題だけにしているので、それはやめておきますね。プロフィールや仲間のこと、楽しいお出かけの計画なら喜んでお手伝いします。",
		"es": "Aquí mantengo las cosas para todos los públicos, así que eso lo dejo pasar. Encantada de ayudarte con tu perfil, tus compañeros o a organizar una buena quedada.",
	},
	"Thanks for being honest with me. SynAgent is only for adults aged 18 and over, so I can't help you with dating here, and I've paused this account until someone on our team has looked at it. No hard feelings.": {
		"en": "Thanks for being honest with me. SynAgent is only for adults aged 18 and over, so I can't help you find company here, and I've paused this account until someone on our team has looked at it. No hard feelings.",
		"ko": "솔직하게 말해 줘서 고마워요. SynAgent는 만 18세 이상 성인만 이용할 수 있어서 여기서는 함께할 사람 찾기를 도와드릴 수 없고, 팀원이 확인할 때까지 이 계정을 일시 중지했어요. 기분 나쁘게 생각하지 말아요.",
		"ja": "正直に話してくれてありがとう。SynAgent は 18 歳以上の成人専用なので、ここで仲間探しのお手伝いはできません。チームの担当者が確認するまで、このアカウントは一時停止しました。悪く思わないでくださいね。",
		"es": "Gracias por ser sincero conmigo. SynAgent es solo para adultos de 18 años o más, así que no puedo ayudarte a encontrar compañía aquí, y he pausado esta cuenta hasta que alguien de nuestro equipo la revise. Sin rencores.",
	},
	"Feedback after dates privately shapes who I introduce you to, and those details stay private, so there's no number for you to look at. The privacy policy explains more. So tell me: what made your favorite date your favorite?": {
		"en": "Feedback after meetups privately shapes who I introduce you to, and those details stay private, so there's no number for you to look at. The privacy policy explains more. So tell me: what made your favorite meetup your favorite?",
		"ko": "모임 후 피드백은 제가 다음에 누구를 소개할지에 비공개로 영향을 줘요. 자세한 내용은 비공개라서 보여 드릴 숫자는 없어요. 더 알고 싶다면 개인정보 처리방침을 참고해 주세요. 그럼 말해 주세요. 가장 좋았던 모임은 뭐가 좋았나요?",
		"ja": "お出かけの後のフィードバックは、次に誰を紹介するかに非公開で影響します。詳細は非公開なので、お見せできる数字はありません。詳しくはプライバシーポリシーをご覧ください。では教えてください。一番よかったお出かけは、どこがよかったですか？",
		"es": "Los comentarios después de cada quedada influyen en privado en a quién te presento, y esos detalles son privados, así que no hay ningún número que puedas ver. La política de privacidad explica más. Cuéntame: ¿qué hizo que tu quedada favorita fuera tu favorita?",
	},
	"Feedback after dates privately shapes who I introduce you to, and those details stay private. The privacy policy explains more. Tell me more about what you're looking for, and I'll keep it in mind for your next introduction.": {
		"en": "Feedback after meetups privately shapes who I introduce you to, and those details stay private. The privacy policy explains more. Tell me more about the kind of company you're looking for, and I'll keep it in mind for your next introduction.",
		"ko": "모임 후 피드백은 제가 다음에 누구를 소개할지에 비공개로 영향을 주고, 자세한 내용은 비공개예요. 더 알고 싶다면 개인정보 처리방침을 참고해 주세요. 어떤 사람과 함께하고 싶은지 더 들려주면 다음 소개 때 기억해 둘게요.",
		"ja": "お出かけの後のフィードバックは、次に誰を紹介するかに非公開で影響し、詳細は非公開です。詳しくはプライバシーポリシーをご覧ください。どんな仲間を探しているかもっと教えてくれたら、次の紹介で覚えておきますね。",
		"es": "Los comentarios después de cada quedada influyen en privado en a quién te presento, y esos detalles son privados. La política de privacidad explica más. Cuéntame más sobre la compañía que buscas y lo tendré en cuenta para tu próxima presentación.",
	},
	"Please don't send money, gift cards or crypto to anyone you've met here. It's the classic sign of a scam, however good the story sounds, and real matches don't ask. I've flagged it for our safety team. Do you want to block them?": {
		"en": "Please don't send money, gift cards or crypto to anyone you've met here. It's the classic sign of a scam, however good the story sounds, and people who genuinely want your company don't ask. I've flagged it for our safety team. Do you want to block them?",
		"ko": "여기서 만난 사람에게 돈, 기프트 카드, 암호화폐를 보내지 마세요. 아무리 그럴듯한 이야기여도 전형적인 사기 신호이고, 진심으로 함께하려는 사람은 그런 걸 요구하지 않아요. 안전팀에 알렸어요. 차단할까요?",
		"ja": "ここで出会った人に、お金やギフトカード、暗号資産を送らないでください。どんなにもっともらしい話でも典型的な詐欺のサインで、本当の仲間はそんなことを頼みません。安全チームに知らせました。ブロックしますか？",
		"es": "Por favor, no envíes dinero, tarjetas regalo ni criptomonedas a nadie que hayas conocido aquí. Es la señal clásica de una estafa, por muy buena que suene la historia, y quien de verdad quiere tu compañía no lo pide. Ya avisé a nuestro equipo de seguridad. ¿Quieres bloquear a esa persona?",
	},
	"You're out of credits, so I can't answer this one yet. Your first match was free, and now each message to me costs a credit. Top up and we'll pick up right here. Anything about your safety or your account is always free.": {
		"en": "You're out of credits, so I can't answer this one yet. Your first introduction was free, and now each message to me costs a credit. Top up and we'll pick up right here. Anything about your safety or your account is always free.",
		"ko": "크레딧이 다 떨어져서 이번 메시지엔 아직 답할 수 없어요. 첫 소개는 무료였고, 이제 저에게 보내는 메시지마다 1크레딧이 들어요. 충전하면 여기서 이어서 얘기해요. 안전이나 계정에 관한 건 언제나 무료예요.",
		"ja": "クレジットがなくなったので、このメッセージにはまだ答えられません。最初の紹介は無料で、これからは私へのメッセージごとに 1 クレジットかかります。チャージしたら、ここから続けましょう。安全やアカウントのことはいつでも無料です。",
		"es": "Te has quedado sin créditos, así que aún no puedo responder a esto. Tu primera presentación era gratis, y ahora cada mensaje para mí cuesta un crédito. Recarga y seguimos aquí mismo. Lo relacionado con tu seguridad o tu cuenta siempre es gratis.",
	},
	"This date plan was cancelled.": {
		"en": "This meetup plan was cancelled.",
		"ko": "이 모임 계획은 취소됐어요.",
		"ja": "このお出かけの計画はキャンセルされました。",
		"es": "Este plan de quedada se ha cancelado.",
	},
	"It's a match!": {
		"en": "You've teamed up!",
		"ko": "함께하게 됐어요!",
		"ja": "仲間になりました！",
		"es": "¡Ya formáis equipo!",
	},
	"Date ideas are ready": {
		"en": "Meetup spots are ready",
		"ko": "만날 장소가 준비됐어요",
		"ja": "お出かけ先の候補ができました",
		"es": "Ya tienes sitios para la quedada",
	},
	"A time for your date": {
		"en": "A time for your meetup",
		"ko": "모임 시간 제안",
		"ja": "お出かけの時間",
		"es": "Una hora para vuestra quedada",
	},
	"Your date is confirmed": {
		"en": "Your meetup is confirmed",
		"ko": "모임이 확정됐어요",
		"ja": "お出かけが確定しました",
		"es": "Tu quedada está confirmada",
	},
	"Your date is tomorrow": {
		"en": "Your meetup is tomorrow",
		"ko": "내일이 모임이에요",
		"ja": "お出かけは明日です",
		"es": "Tu quedada es mañana",
	},
	"Your date is coming up": {
		"en": "Your meetup is coming up",
		"ko": "모임이 곧이에요",
		"ja": "もうすぐお出かけです",
		"es": "Tu quedada está cerca",
	},
}
