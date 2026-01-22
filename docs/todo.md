
priority ui fixes 

FE redo goal and templates 
FE add goal logging 
check completion of goal and task, does task completion is needed for goals
completion, streak etc check and ui changes, also ui changes in goals based on tyep of goal 
gial craation anf all 
goal chils, goals and tempaltes i nitmeline and agenda 





optimise file structure, db queries, api calls and fe components setup 


optimise emotion generation as it fucks up on every load 


check all integration and use cases , all goal type , template etc
ceheck all tranition, categories, prioritues etc


Fe extra items and quick logs
fe analytics + be analytics 









footer is not sticky to bottom and there is padding on both end of footer why ??????




for posive and negtives how to set emotion 
FIX task ui, make it look better, expand the rich text, move header back and breadcrumb to page header search and remove searchits not require in taks create and edit keep it only for dahboard 



in timeline or agnda the emotion is nto showing 
cor completed tasks the timelien should shoiw a different ui its not changing ?




also how are you showing the infered emotion , show both infered and selected in smae line 



double click should select with confirmation for emotion 




clean emotions page and not required code realted to emotion, unused code or page that is not used 









clean up frontend code fix linting errors and warning, move constants to seperate file, move interfaces to seperate file. add comments 
make utils where required, make components for reusing items 
follow and expand design language .md if required 







Modals are not rendering over sidebar 

when the hover is based on pan center and then user hovers on some other emotion the pan center item hover flickers, is there any solution fort his ?

make side bar more pretty with anuimation and more cleanup 

fix the username, email throughout and fix edit properly ----------
timezone throught based on 

SET default states in setting for : completed staate, last task, from last task 
set default time for retro and based ont hat update timeline with default time zone there as well based on user proefernece in case by defaulpick local 

etc 



lets use 
 vis-timeline


use pnpm 

migrate timeline to some package where its easy for me to handle blow things giving me good customisibaility on tasks ui as well i want opensource package , keeping the timeline layout as it it right now 


make ui more beautiful of timeline
add drag and drop option 
add checkbox to complete to task modal and to the ttask ui 

add option ot complete in timeline 
add confirmation of removing complete  (if not already created generalised confirmation modal that cna be customised for diffrent uses )

write code such its easy to work with by other devs and is expandable for more ui changes like task template and goals etc 




fix the color of tasks without any category  
fix the task ui



add websocket infinite scroll + text heighlighting in search  for task table 
make the setup generalised such that it can be expanded and uded for other items in future 




add option ot complete in timeline 
add option to complete in modal 
add confirmation of removing complete 

add confirmation od set complete in timeline

make complete tick better in task ui 

end of day based on retro time 


add search on category + better category 

fix login ui 


emotion 




3. Best Timeline Packages for Infinite Customization
Here are the top options:

Package	Pros	Cons
vis.js Timeline	Built-in overlap stacking, highly customizable CSS, handles large datasets, zoom/pan, mature library	Vanilla JS (needs wrapper), larger bundle size
SVAR Svelte Gantt	Native Svelte, drag-and-drop, task resizing, active development	More Gantt-focused than pure timeline
Schedule-X	Native Svelte component, highly customizable, designed for calendar views	More calendar-focused
svelte-gantt	Lightweight, native Svelte, interactive	Some issues with specific durations
My Recommendation: For your use case (horizontal task timeline with overlap handling), vis.js Timeline is the most robust choice. It has explicit overlap handling that stacks items intelligently and is infinitely customizable via CSS and options.





Goals/Habbits
Values
Retro

make category unique by color 

Task Template
Flomodoro + subtask 







cjeck and remove all 
Period-end snapshots are created automatically:
Weekly goals: Snapshot on Monday at 00:01 UTC
Monthly goals: Snapshot on the 1st of each month at 00:01 UTC

and other scheduled, fix it 
















use daisy ui timeline to show it in history  tab 

also detail out the tasks tab, add filter etc 
for hostory allow with task and without tasks 

make ui for goals, and goal modal prettier, cleaner and more user friendly , easy to use and easy to understand 





check goal tasks remove it from goal  main api, get api , list api and make it to sepete api  with more task related details to show in fe 
make sure there is a goal logs api to fetch all logs for goals logs and snapshot history for histortyy view 
add start date for goals
remove linked goals from get task list api and get task api and create speerate pai for linekd goals if not exsiting 
also make sure ther eis crud for unit 







the modal is not showing  start date , end date, unti etc ?  
alsothe 







better goal logs in genral check all 
update tasks tab in goal modal 







fix time timeline next




















the modal is not showing  start date , end date, unti etc ?  








create cub components for all huge components where ever its sensable throughtou the frontend 







timezone settings throughtout --- > need to change 
check created_from ?? what used ??also check design tab in surrealist


on goal update only add to logs if somethign changes and mention what updated , also mention more dratils of ehat all is target etc ------


















Current Streak Current Value all too bug in history 




Add goasl in agenda view 

fix goals in tasks 

fix goals filter 

check al lgoal modal items 

wrok on templates 




add empty stats for agenda tasks --------------------------------------