// resolveSubmitDate decides which calendar day the entry form uses. A
// brand-new expense whose date the user never touched always follows the real
// current day (`now`), so a form left open across an iOS overnight background
// freeze — where `pickedDate` is frozen at the day the form was first opened —
// can't silently show or store yesterday's date. Edits and explicit user picks
// are returned as-is.
//
// Used in two places: with the reactive `today` to drive the date pill's
// displayed value, and with a fresh `new Date()` at submit time as the
// write-time guarantee.
export function resolveSubmitDate(
  isEdit: boolean,
  dateTouched: boolean,
  pickedDate: Date,
  now: Date,
): Date {
  return !isEdit && !dateTouched ? now : pickedDate;
}
