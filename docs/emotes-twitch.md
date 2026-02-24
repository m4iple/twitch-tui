
emots from twitch are easyest - they are in the message struct inclded

exaple whats in the emotes array
emotes: [{"Name":"meisakStare","ID":"emotesv2_57a5f10f19e54747acfad54741f9d215","Count":1,"Positions":[{"Start":0,"End":10}]}]

if we get one 
    - call the cdn to get it
    - format it to for the terminal
    - async then dispaly it when ready
    - cache it so its faster on next use

- cache is sesson depended - on program close we clear it
