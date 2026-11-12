// An untouched new expense follows `now`: after an iOS overnight freeze
// `pickedDate` is still the day the form was opened.
export function resolveSubmitDate(
  isEdit: boolean,
  dateTouched: boolean,
  pickedDate: Date,
  now: Date,
): Date {
  return !isEdit && !dateTouched ? now : pickedDate;
}
