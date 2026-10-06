# CLI Epics Fix

Now first, please fix cli so that everything works:

## EpicsListScreen
- both active and inactive epics are listed
- Paint active epics in Foreground color and inactive epics in AccentRed color
- <del> deletes epic if it is not attached to any change - if it is switch to inactive
- <space> toggles active/inactive state
- use same layout as on ChangesListScreen - vertically allign all the columns
- ID is 4 chars wide and aligned to the right with id values bellow
- Name is 30 chars wide aligned to the left
- DoneTC is 6 chars wide and aligned to the right
- Compl is 5 chars wide and aligned to the right, values are printed with percentage sign like this: 100% 33%...
- Chngs is 5 chars wide and aligned to the right
- Active is aligned to the left and only outputing value `inactive` when epic is set inactive

On ChangeDetailsScreen when user selects epic for the change only active epics must be listed.

## ChangeListScreen
- only active changes are listed by default:
- /del-change menu item is inserted just bellow /new-change - when user switches to /filter-inactive mode this item becomes /undel-change
- add menu item /inactive-filter just bellow find filter
- /inactive-filter label is added on top of screen, on the right of /find-filter
- selecting /inactive-filter toggles active/inactive changes list
- when /inactive-filter is selected paint it in AccentRed color and select only inactive changes
- when /inactive-filter is again selected it gets painted in gray color as other filters in the line
- change cannot be deleted - when <del> is clicked or /del-change is selected on the change - prompt Are you sure? Yes/No is shown in bottom prompt - if yes is clicked change is switched to inactive
- user can undo set inactive by clicking <space> or selecting /undel-change from the menu
