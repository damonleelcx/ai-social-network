// What a profile says, in words. Words only, never a score.
const { t } = require('./i18n');

const GOALS = { long_term: 'Long-term', marriage: 'Marriage', short_term: 'Short-term', casual: 'Casual', friends_first: 'Friends first', not_sure: 'Not sure yet' };
const HABITS = { no: 'No', sometimes: 'Sometimes', yes: 'Yes' };
const KIDS = { none: "Don't have kids", have: 'Have kids' };
const WANTS_KIDS = { yes: 'Want kids', no: "Don't want kids", open: 'Open to it' };

const label = (table, value) => (table[value] ? t(table[value]) : '');

function line(p) {
  return [p.age ? String(p.age) : '', p.pronouns, p.city_name].filter(Boolean).join(' · ');
}

function details(p) {
  return [
    ['Looking for', label(GOALS, p.goal)],
    ['Work', p.occupation],
    ['Studied', p.education],
    ['Height', p.height_cm ? t('{name} cm', { name: p.height_cm }) : ''],
    ['Smokes', label(HABITS, p.smoking)],
    ['Drinks', label(HABITS, p.drinking)],
    ['Kids', label(KIDS, p.kids)],
    ['Future kids', label(WANTS_KIDS, p.wants_kids)],
    ['Speaks', (p.languages || []).join(', ')],
  ].filter((d) => d[1]).map((d) => ({ key: t(d[0]), value: d[1] }));
}

// For the forms that set these, in the order they're listed above.
const options = (table) => Object.keys(table).map((value) => ({ value, label: t(table[value]) }));

module.exports = { line, details, options, GOALS, HABITS, KIDS, WANTS_KIDS };
