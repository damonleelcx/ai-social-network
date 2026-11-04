// The landing page's translations, keyed by the English text. English is the default and needs
// no entry: anything without one stays as it's written in the page. Legal pages stay in English.

const STRINGS = {
  // [Simplified Chinese, Korean, Japanese]
  'SynAgent: a matchmaker you talk to, not a deck you swipe': ['SynAgent：一位能聊天的红娘，不是一叠划来划去的卡片', 'SynAgent: 넘기는 카드가 아니라, 대화하는 매치메이커', 'SynAgent：スワイプするカードではなく、話せるマッチメイカー'],
  'SynAgent is a dating app built around Suki, an AI matchmaker. Tell her about you and she introduces you to people in your city. No swiping.': ['SynAgent 是一款以 AI 红娘 Suki 为核心的交友应用。告诉她你的故事，她会为你介绍同城的人。无需滑动。', 'SynAgent는 AI 매치메이커 Suki를 중심으로 한 데이팅 앱입니다. 당신에 대해 이야기하면 같은 도시의 사람들을 소개해 드려요. 스와이프는 없어요.', 'SynAgent は AI マッチメイカー Suki を中心にしたデートアプリです。あなたのことを話すと、同じ街の人を紹介してくれます。スワイプはありません。'],
  'Skip to how it works': ['跳到使用方式', '이용 방법으로 건너뛰기', '使い方へ移動'],
  'SynAgent home': ['SynAgent 首页', 'SynAgent 홈', 'SynAgent ホーム'],
  Sections: ['页面导航', '섹션', 'セクション'],
  Language: ['语言', '언어', '言語'],
  'How it works': ['使用方式', '이용 방법', '使い方'],
  'Why Suki': ['为什么选 Suki', '왜 Suki인가', 'Suki を選ぶ理由'],
  'Your control': ['你说了算', '당신의 선택', 'あなたが決める'],
  Download: ['下载', '다운로드', 'ダウンロード'],
  Light: ['浅色', '라이트', 'ライト'],
  Dark: ['深色', '다크', 'ダーク'],
  'Switch to dark mode': ['切换到深色模式', '다크 모드로 전환', 'ダークモードに切り替え'],
  'Switch to light mode': ['切换到浅色模式', '라이트 모드로 전환', 'ライトモードに切り替え'],

  'A matchmaker you talk to,': ['一位能聊天的红娘，', '넘기는 카드가 아니라,', 'スワイプするカードではなく、'],
  'not a deck you swipe.': ['不是一叠划来划去的卡片。', '대화하는 매치메이커.', '話せるマッチメイカー。'],
  'No swiping. No endless decks.': ['不用滑动，没有刷不完的卡片。', '스와이프도, 끝없는 카드도 없어요.', 'スワイプも、終わりのないカードもなし。'],
  'Just someone worth meeting.': ['只有值得见的人。', '만날 만한 사람만 있어요.', '会う価値のある人だけ。'],
  'Your next': ['你的下一段', '다음 이야기는', '次の物語は'],
  'story starts': ['故事', '한마디', 'ひとことの'],
  'with hello': ['从你好开始', '인사로 시작돼요', 'あいさつから'],
  'Download the app': ['下载应用', '앱 다운로드', 'アプリをダウンロード'],
  'For adults 18 and over.': ['仅限 18 岁及以上成年人。', '만 18세 이상 성인 전용입니다.', '18歳以上の成人向けです。'],

  'Meet Suki': ['认识 Suki', 'Suki를 만나 보세요', 'Suki に会う'],
  'Suki in pixel art: long dark hair, a glowing cat hair clip, winking and flashing a peace sign': ['像素风的 Suki：深色长发，戴着发光的猫咪发夹，眨着眼比出剪刀手', '픽셀 아트로 그린 Suki: 긴 검은 머리에 빛나는 고양이 머리핀을 하고, 윙크하며 브이를 하고 있어요', 'ピクセルアートの Suki：長い黒髪に光る猫のヘアクリップ、ウインクしてピースサイン'],
  'Playful · Curious · Your AI wingwoman': ['爱玩 · 好奇 · 你的 AI 僚机', '장난기 · 호기심 · 당신의 AI 윙우먼', 'おちゃめ · 好奇心旺盛 · あなたの AI ウィングウーマン'],
  '"Tell me about you, and I\'ll introduce you to people worth meeting."': ['“跟我说说你吧，我会介绍值得见的人给你。”', '“당신에 대해 들려주세요. 만날 만한 사람을 소개해 드릴게요.”', '「あなたのことを教えて。会う価値のある人を紹介するね。」'],
  Talk: ['聊聊', '대화', '話す'],
  'Why her?': ['为什么是她？', '왜 그녀일까요?', 'なぜ彼女？'],
  "* It's a good day to meet someone.": ['* 今天很适合认识新朋友。', '* 누군가를 만나기 좋은 날이에요.', '* 今日は誰かに出会うのにいい日だね。'],
  "* You'd be surprised what a real conversation can do.": ['* 一场真正的对话，能带来你想不到的改变。', '* 진짜 대화가 얼마나 큰 힘이 있는지 알면 놀랄 거예요.', '* 本当の会話の力に、きっと驚くよ。'],
  "* I won't show you a deck. I'll just tell you about someone.": ['* 我不会丢给你一叠卡片，只会跟你聊聊某个人。', '* 카드 더미는 안 보여 줘요. 그냥 어떤 사람 이야기를 들려줄게요.', '* カードの山は見せないよ。ただ、ある人の話をするね。'],
  "* It's only a match when you both say yes.": ['* 只有你们都说好，才算配对成功。', '* 두 사람 모두 좋다고 해야 매치가 돼요.', '* ふたりともイエスと言ったときだけ、マッチになるよ。'],
  Meet: ['相见', '만남', '出会う'],
  Date: ['约会', '데이트', 'デート'],
  Learn: ['了解', '배움', '学ぶ'],

  'Someone she might bring you': ['她可能为你介绍的人', 'Suki가 소개할지도 모를 사람', 'Suki が紹介するかもしれない人'],
  "Sample introductions, not real people. Suki explains each one in words. There's never a score.": ['这些是示例介绍，并非真实人物。Suki 会用文字说明每一位，从不打分。', '실제 인물이 아닌 소개 예시입니다. Suki는 한 사람 한 사람을 말로 설명해요. 점수는 절대 없어요.', 'これは紹介の例で、実在の人物ではありません。Suki は一人ひとりを言葉で説明します。点数は一切ありません。'],
  "* Say yes or pass. It's only a match when you both say yes.": ['* 说好或跳过。只有你们都说好，才算配对成功。', '* 좋아요 또는 패스를 골라요. 두 사람 모두 좋다고 해야 매치가 돼요.', '* イエスかパスを選んで。ふたりともイエスのときだけマッチになるよ。'],
  'Climbs on Sundays, argues about ramen. You said you want someone curious.': ['周日去攀岩，爱争论哪家拉面最好。你说过想找个有好奇心的人。', '일요일엔 클라이밍을 하고, 라멘 얘기엔 진심이에요. 호기심 많은 사람을 원한다고 했죠.', '日曜はクライミング、ラーメン談義に本気。好奇心のある人がいいって言ってたよね。'],
  'Bakes badly, reads a lot. You both think a walk beats a bar.': ['烘焙很烂，书读得很多。你们都觉得散步比泡吧好。', '빵은 못 굽지만 책은 많이 읽어요. 둘 다 술집보다 산책을 좋아하죠.', 'お菓子作りは下手だけど読書家。ふたりともバーより散歩派だよ。'],
  'Sings off-key on purpose. You told me you laugh at bad puns. Trust me on this.': ['故意唱跑调。你说过冷笑话能逗笑你。这个就信我吧。', '일부러 음을 틀리게 불러요. 썰렁한 말장난에 웃는다고 했죠. 이번엔 저를 믿어 봐요.', 'わざと音を外して歌う人。ダジャレで笑うって言ってたよね。ここは信じて。'],
  "Draws maps of places that don't exist. You both want to travel slowly.": ['爱画不存在的地方的地图。你们都想慢慢旅行。', '존재하지 않는 곳의 지도를 그려요. 둘 다 천천히 여행하고 싶어 하죠.', '存在しない場所の地図を描く人。ふたりとも、ゆっくり旅をしたいタイプだよ。'],
  'Runs a tiny record shop. You both still buy things just for the sleeve art.': ['开着一家小唱片店。你们都还会为了封面买唱片。', '작은 음반 가게를 운영해요. 둘 다 아직도 커버 아트 때문에 음반을 사죠.', '小さなレコード店を営む人。ふたりともジャケット目当てで買っちゃうタイプだよ。'],
  'Pass on {name}': ['跳过 {name}', '{name} 패스', '{name} をパス'],
  'Say yes to {name}': ['对 {name} 说好', '{name}에게 좋아요', '{name} にイエス'],
  "* Nice. If {name} says yes too, you'll start talking.": ['* 不错。如果 {name} 也说好，你们就可以开始聊天了。', '* 좋아요. {name}도 좋다고 하면 대화를 시작할 수 있어요.', '* いいね。{name} もイエスなら、話し始められるよ。'],
  "* Noted. Passing tells me something too. I'll learn from it.": ['* 记下了。跳过也能告诉我一些事，我会从中学习。', '* 알겠어요. 패스도 제게 힌트가 돼요. 참고할게요.', '* わかった。パスも大事なヒントだから、ちゃんと覚えておくね。'],

  'I · How it works': ['一 · 使用方式', 'I · 이용 방법', 'I · 使い方'],
  'One matchmaker.': ['一位红娘，', '매치메이커 한 명,', 'マッチメイカーはひとり。'],
  'Five small steps.': ['五个小步骤。', '작은 다섯 걸음.', '小さな5ステップ。'],
  'Keep scrolling, or pick a step on her screen.': ['继续滚动，或在她的屏幕上选择一个步骤。', '계속 스크롤하거나 화면에서 단계를 골라 보세요.', 'スクロールを続けるか、画面でステップを選んでください。'],
  'Status:': ['状态：', '상태:', '状態：'],
  'Swipes: 0': ['滑动次数：0', '스와이프: 0', 'スワイプ：0'],
  'Suki ·': ['Suki ·', 'Suki ·', 'Suki ·'],
  listening: ['倾听中', '듣는 중', '聞いています'],
  searching: ['寻找中', '찾는 중', '探しています'],
  coaching: ['指导中', '코칭 중', 'サポート中'],
  scouting: ['选址中', '장소 찾는 중', '場所探し中'],
  learning: ['学习中', '배우는 중', '学習中'],
  'Your type, in her words': ['用她的话说，你的类型', 'Suki의 말로 보는 당신의 타입', '彼女の言葉で語る、あなたのタイプ'],
  Sample: ['示例', '예시', '例'],
  'Laughs at bad puns': ['会被冷笑话逗笑', '썰렁한 말장난에 웃어요', 'ダジャレで笑う'],
  'Prefers mornings': ['喜欢早晨', '아침을 좋아해요', '朝が好き'],
  'Wants someone curious': ['想找有好奇心的人', '호기심 많은 사람을 원해요', '好奇心のある人がいい'],
  'Step {n}': ['第 {n} 步', '{n}단계', 'ステップ {n}'],
  'Get to know Suki': ['认识 Suki', 'Suki 알아가기', 'Suki を知る'],
  'Get introduced': ['获得介绍', '소개받기', '紹介してもらう'],
  'Start talking': ['开始聊天', '대화 시작하기', '話し始める'],
  'Plan the date': ['安排约会', '데이트 계획하기', 'デートを計画する'],
  'Tell her how it went': ['告诉她进展如何', '어땠는지 들려주기', 'どうだったか伝える'],
  "Set up your profile by talking to her: who you are, your photos, your interests, and what you're looking for.": ['通过和她聊天来设置个人资料：你是谁、你的照片、你的兴趣，以及你在寻找什么。', 'Suki와 대화하며 프로필을 만들어요. 당신이 어떤 사람인지, 사진, 관심사, 찾는 사람까지요.', '彼女と話しながらプロフィールを作ります。あなたのこと、写真、興味、探している相手について。'],
  "She looks for people in your city who suit you, and explains why she thinks you'd get along. It's only a match when you both say yes.": ['她会在你所在的城市寻找适合你的人，并说明为什么觉得你们合得来。只有你们都说好，才算配对成功。', '같은 도시에서 잘 맞을 사람을 찾고, 왜 잘 어울릴지 설명해 줘요. 두 사람 모두 좋다고 해야 매치가 돼요.', '同じ街であなたに合う人を探し、なぜ気が合いそうかを説明します。ふたりともイエスのときだけマッチになります。'],
  'Chat with your match directly. She can suggest an opener or a question when things go quiet, and you can turn that off.': ['直接和配对对象聊天。冷场时她可以建议开场白或问题，你也可以关闭这项功能。', '매치된 상대와 직접 대화해요. 대화가 뜸해지면 첫마디나 질문을 제안해 주고, 이 기능은 끌 수 있어요.', 'マッチした相手と直接チャットします。会話が止まったら話題や質問を提案してくれます。この機能はオフにできます。'],
  'She shortlists public places based on what you both enjoy. You each pick a favorite privately, and the date is set once you both confirm.': ['她会根据你们共同的喜好筛选公共场所。你们各自私下选出最喜欢的地方，双方确认后约会就定下来了。', '두 사람이 좋아하는 것을 바탕으로 공공장소를 추려 줘요. 각자 마음에 드는 곳을 비공개로 고르고, 둘 다 확인하면 데이트가 확정돼요.', 'ふたりの好みに合わせて公共の場所を絞り込みます。それぞれ非公開でお気に入りを選び、ふたりが確認するとデートが決まります。'],
  'What you share after a date helps her make better introductions next time.': ['约会后你分享的内容，能帮她下次做出更好的介绍。', '데이트 후 들려준 이야기는 다음 소개를 더 좋게 만들어 줘요.', 'デートの後に話してくれたことが、次の紹介をより良くします。'],

  'II · Why Suki is different': ['二 · Suki 有何不同', 'II · Suki가 다른 이유', 'II · Suki が違う理由'],
  'Not an algorithm with a chat box': ['不只是带聊天框的算法', '채팅창 달린 알고리즘이 아니에요', 'チャット欄付きのアルゴリズムではありません'],
  'She has a personality': ['她有自己的个性', '개성이 있어요', '個性があります'],
  'Suki has her own character, face and voice, and she answers as she thinks instead of making you wait. You can choose how she talks to you.': ['Suki 有自己的性格、面容和声音，边想边回答，不会让你干等。你还可以选择她和你说话的方式。', 'Suki에게는 자신만의 성격과 얼굴, 목소리가 있고, 기다리게 하지 않고 생각하는 대로 답해요. 어떻게 말할지도 고를 수 있어요.', 'Suki には自分の性格、顔、声があり、待たせることなく考えながら答えます。話し方も選べます。'],
  'She learns your type': ['她了解你的类型', '당신의 타입을 알아가요', 'あなたのタイプを学びます'],
  'Her introductions improve with what you accept, what you pass on, and what you say after your dates.': ['你接受了谁、跳过了谁、约会后说了什么，都会让她的介绍越来越好。', '수락한 사람, 패스한 사람, 데이트 후 들려준 이야기로 소개가 점점 좋아져요.', '受け入れた人、パスした人、デート後に話したことをもとに、紹介が良くなっていきます。'],
  'She keeps things respectful': ['她维护尊重的氛围', '서로 존중하도록 지켜요', '敬意のある場を保ちます'],
  'Explicit photos, messages and profile text are filtered, and you can block or report anyone.': ['露骨的照片、消息和资料文字都会被过滤，你也可以屏蔽或举报任何人。', '노골적인 사진, 메시지, 프로필 문구는 걸러지고, 누구든 차단하거나 신고할 수 있어요.', '露骨な写真、メッセージ、プロフィール文はフィルタリングされ、誰でもブロックや通報ができます。'],
  'She finds the place': ['她帮你找地方', '장소를 찾아 줘요', '場所を探します'],
  "She shortlists public places you'd both enjoy. You each pick a favorite privately, and the date is set once you both confirm.": ['她会筛选你们都会喜欢的公共场所。你们各自私下选出最喜欢的地方，双方确认后约会就定下来了。', '두 사람 모두 즐길 만한 공공장소를 추려 줘요. 각자 비공개로 고르고, 둘 다 확인하면 데이트가 확정돼요.', 'ふたりが楽しめそうな公共の場所を絞り込みます。それぞれ非公開で選び、ふたりが確認するとデートが決まります。'],

  'III · Your control': ['三 · 你说了算', 'III · 당신의 선택', 'III · あなたが決める'],
  'You hold the save file': ['存档在你手里', '세이브 파일은 당신 것', 'セーブデータはあなたのもの'],
  'Suki is an AI, not a person. What she remembers belongs to you.': ['Suki 是 AI，不是真人。她记住的内容属于你。', 'Suki는 사람이 아니라 AI예요. Suki가 기억하는 내용은 당신의 것이에요.', 'Suki は人ではなく AI です。彼女が覚えていることは、あなたのものです。'],
  Read: ['查看', '보기', '見る'],
  'what Suki remembers about you.': ['Suki 记住的关于你的内容。', 'Suki가 당신에 대해 기억하는 내용을 볼 수 있어요.', 'Suki があなたについて覚えていることを確認できます。'],
  Delete: ['删除', '삭제', '削除'],
  'any of it, or reset what she has learned.': ['其中任何内容，或重置她学到的一切。', '원하는 내용을 지우거나, 배운 것을 초기화할 수 있어요.', 'どの内容も消せます。学習内容のリセットもできます。'],
  Export: ['导出', '내보내기', '書き出し'],
  'your data whenever you like.': ['你的数据，随时都可以。', '언제든 데이터를 내보낼 수 있어요.', 'データをいつでも書き出せます。'],
  Leave: ['离开', '떠나기', '退会'],
  'by deleting your account in Settings.': ['在设置中删除账户即可。', '설정에서 계정을 삭제할 수 있어요.', '設定からアカウントを削除できます。'],

  'IV · Download': ['四 · 下载', 'IV · 다운로드', 'IV · ダウンロード'],
  'Take her with you': ['把她带在身边', 'Suki와 함께하세요', '彼女を連れていこう'],
  'Open the file on your phone. Android will ask you to allow installs from your browser the first time.': ['在手机上打开文件。首次安装时，Android 会询问是否允许通过浏览器安装。', '휴대폰에서 파일을 여세요. 처음에는 Android가 브라우저에서 설치를 허용할지 물어봐요.', 'スマートフォンでファイルを開いてください。初回は、ブラウザからのインストールを許可するか Android が確認します。'],
  "SynAgent isn't in the App Store or Google Play yet, so open it in your browser or use a direct download. For adults 18 and over.": ['SynAgent 还没有上架 App Store 和 Google Play，你可以直接在浏览器里打开，或者下载安装。仅限 18 岁及以上。', 'SynAgent는 아직 App Store와 Google Play에 없어요. 브라우저에서 열거나 직접 내려받아 주세요. 18세 이상만 이용할 수 있어요.', 'SynAgent はまだ App Store や Google Play にありません。ブラウザで開くか、直接ダウンロードしてください。18 歳以上の方が対象です。'],
  'In your browser': ['在浏览器里用', '브라우저에서', 'ブラウザで'],
  'Nothing to install, and it works on a computer too. Your account is the same one the app uses.': ['不用安装，电脑上也能用。账号和 App 是同一个。', '설치할 필요 없고 컴퓨터에서도 돼요. 계정은 앱과 같아요.', 'インストール不要で、パソコンでも使えます。アカウントはアプリと共通です。'],
  'Open SynAgent': ['打开 SynAgent', 'SynAgent 열기', 'SynAgent を開く'],
  'Not only dates': ['不只是约会', '데이트만이 아니에요', 'デートだけじゃない'],
  "Nobody to see a movie with tonight? Post a plan, for a meal, a trip or a business contact too. People in your city say they're in, and you pick who to go with.": ['今晚想看电影却没人陪？发一个搭子请求吧，吃饭、旅行、商务合作都可以。同城的人说“我想去”，你来选和谁一起。', '오늘 밤 함께 영화 볼 사람이 없나요? 계획을 올려 보세요. 식사, 여행, 비즈니스 인맥도 좋아요. 같은 도시 사람들이 함께하겠다고 하면 당신이 고르면 돼요.', '今夜一緒に映画を観る人がいない？予定を投稿しましょう。ごはん、旅行、ビジネスのつながりも OK。同じ街の人が手を挙げたら、一緒に行く人を選ぶだけ。'],
  'Log in': ['登录', '로그인', 'ログイン'],
  "Tell Suki about you. She introduces you to people in your city to date, or finds you company for a movie, a trip or a business contact.": ['告诉 Suki 你是什么样的人。她会介绍同城的人和你约会，也能帮你找看电影、旅行、商务合作的搭子。', 'Suki에게 당신에 대해 알려 주세요. 같은 도시에서 데이트할 사람을 소개하고, 영화나 여행, 비즈니스 인맥을 함께할 사람도 찾아 줘요.', 'Suki にあなたのことを話してください。同じ街でデートの相手を紹介し、映画や旅行、ビジネスのつながりの仲間も見つけてくれます。'],
  'Most apps hand you a deck. SynAgent gives you someone to talk to, who gets a better sense of your type each time.': ['大多数应用只给你一叠卡片。SynAgent 给你的是一个能聊天的人，而且她每次都更懂你喜欢什么样的人。', '대부분의 앱은 카드 더미를 건네죠. SynAgent는 이야기할 상대를 줘요. 그리고 그녀는 갈수록 당신의 취향을 더 잘 알아가요.', '多くのアプリはカードの山を渡すだけ。SynAgent には話せる相手がいて、話すたびにあなたの好みを分かっていきます。'],
  'Open in your browser': ['在浏览器中打开', '브라우저에서 열기', 'ブラウザで開く'],
  'In your browser, or by direct download.': ['可以直接用浏览器，也可以下载安装。', '브라우저에서 바로 쓰거나 직접 내려받으세요.', 'ブラウザで使うか、直接ダウンロードできます。'],
  'Download for Android': ['下载 Android 版', 'Android용 다운로드', 'Android 版をダウンロード'],
  "This file is not signed, so an iPhone won't install it as it is.": ['此文件未经签名，iPhone 无法直接安装。', '이 파일은 서명되지 않아 iPhone에 그대로 설치할 수 없어요.', 'このファイルは署名されていないため、そのままでは iPhone にインストールできません。'],
  'You need a sideloading tool such as AltStore or Sideloadly, which signs it with your own Apple ID.': ['你需要 AltStore 或 Sideloadly 等侧载工具，用你自己的 Apple ID 为它签名。', 'AltStore나 Sideloadly 같은 사이드로딩 도구로 본인의 Apple ID로 서명해야 해요.', 'AltStore や Sideloadly などのサイドロードツールで、ご自身の Apple ID を使って署名する必要があります。'],
  'Download unsigned IPA': ['下载未签名的 IPA', '서명되지 않은 IPA 다운로드', '未署名の IPA をダウンロード'],
  'Checksums are on the': ['校验值请见', '체크섬은', 'チェックサムは'],
  'release page': ['发布页面', '릴리스 페이지', 'リリースページ'],
  '.': ['。', '에서 확인할 수 있어요.', 'にあります。'],

  'Your move': ['轮到你了', '이제 당신 차례', '次はあなたの番'],
  "Tell Suki about you. She'll take it from there.": ['告诉 Suki 你的故事，剩下的交给她。', 'Suki에게 이야기해 보세요. 나머지는 Suki가 맡을게요.', 'Suki にあなたのことを話して。あとは任せてください。'],
  'Questions, or just want to say hi?': ['有问题，或者只是想打个招呼？', '궁금한 점이 있거나 인사하고 싶으신가요?', '質問がある方も、あいさつだけの方もどうぞ。'],
  'Write to us': ['写信给我们', '문의하기', 'お問い合わせ'],

  "SynAgent is for adults 18 and over. Suki is an AI and can be wrong. Meet in public places and tell someone you trust where you're going.": ['SynAgent 仅限 18 岁及以上成年人使用。Suki 是 AI，可能会出错。请在公共场所见面，并告诉你信任的人你要去哪里。', 'SynAgent는 만 18세 이상 성인 전용입니다. Suki는 AI이므로 틀릴 수 있어요. 공공장소에서 만나고, 믿을 수 있는 사람에게 어디 가는지 알려 주세요.', 'SynAgent は18歳以上の成人向けです。Suki は AI のため、間違えることがあります。公共の場所で会い、信頼できる人に行き先を伝えてください。'],
  Legal: ['法律信息', '법적 고지', '法的情報'],
  Terms: ['条款（英文）', '약관(영문)', '規約（英語）'],
  Privacy: ['隐私（英文）', '개인정보(영문)', 'プライバシー（英語）'],
  'Delete your account': ['删除账户（英文）', '계정 삭제(영문)', 'アカウント削除（英語）'],
  Contact: ['联系我们', '연락처', '連絡先'],
};

const INDEX = { zh: 0, ko: 1, ja: 2 };

export const LANG = document.documentElement.dataset.lang || 'en';

export function tr(english, vars) {
  const entry = STRINGS[english];
  let text = entry && LANG !== 'en' ? entry[INDEX[LANG]] : english;
  for (const [k, v] of Object.entries(vars ?? {})) text = text.replace(`{${k}}`, v);
  return text;
}

export function setLanguage(lang) {
  try {
    localStorage.setItem('synagent.lang', lang);
  } catch {
    // Private windows: the link below still carries it for this page.
  }
  // Fonts, typed lines and cards are all set up once at load, so start clean. A plain URL
  // drops any ?lang= that would override what they just picked.
  location.href = lang === 'en' ? location.pathname : `${location.pathname}?lang=${lang}`;
}

// Everything the page's HTML says, swapped in place, keeping each string's spacing.
export function translatePage() {
  if (LANG === 'en') return;
  document.title = tr(document.title);
  const description = document.querySelector('meta[name="description"]');
  description.content = tr(description.content);
  const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
  for (let node = walker.nextNode(); node; node = walker.nextNode()) {
    if (node.parentElement.closest('script, style, select')) continue;
    const key = node.nodeValue.trim();
    if (!key || !STRINGS[key]) continue;
    node.nodeValue = node.nodeValue.replace(key, tr(key));
  }
  for (const el of document.querySelectorAll('[aria-label], [alt]')) {
    for (const attr of ['aria-label', 'alt']) {
      const value = el.getAttribute(attr);
      if (value && STRINGS[value]) el.setAttribute(attr, tr(value));
    }
  }
  document.documentElement.classList.remove('i18n-pending');
}

// For the build step that trims the fonts: every string in one language.
export const allText = (lang) => Object.values(STRINGS).map((e) => e[INDEX[lang]]).join('');
