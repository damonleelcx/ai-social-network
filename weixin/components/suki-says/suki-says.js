// Empty and loading states are Suki talking, not a grey box. With dots, she's busy.
Component({
  options: { multipleSlots: false },
  properties: {
    face: { type: String, value: 'chibi' },
    size: { type: Number, value: 240 },
    title: { type: String, value: '' },
    body: { type: String, value: '' },
    dots: { type: Boolean, value: false },
  },
});
