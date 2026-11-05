// Suki in her iridescent ring. Sizes are in rpx.
const { face } = require('../../lib/suki');

Component({
  properties: {
    mood: { type: String, value: '' },
    moodAvatar: { type: Boolean, value: true },
    size: { type: Number, value: 80 },
    ring: { type: Number, value: 4 },
  },
  data: { src: face('') },
  observers: {
    'mood, moodAvatar': function update(mood, moodAvatar) {
      this.setData({ src: face(mood, moodAvatar) });
    },
  },
});
