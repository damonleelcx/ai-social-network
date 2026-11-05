// Suki's faces. The same rule as the app: her mood's face, or her portrait when there's no
// mood, an unknown one, or the Avatar moods setting is off.
const FACES = ['portrait', 'happy', 'surprised', 'thinking', 'playful', 'pouty', 'chibi'];

const face = (mood, moodAvatar) => {
  const name = moodAvatar !== false && FACES.indexOf(mood) >= 0 ? mood : 'portrait';
  return `/assets/suki/${name}.jpg`;
};

module.exports = { FACES, face };
