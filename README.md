# Bangalore Vaccine Updater - GoLang CoWIN polling service

This project was used to create the Twitter bot [`Banglore Vaccine Updater`](https://twitter.com/BloreVaccine) to poll CoWIN APIs to detect changes in the vaccination slot availability. And then notify the general public about the same via the above mentioned Twitter bot. 

#### How does it work, and what are the issues and challenges? 

How it works is pretty simple, we make a call to CoWIN APIs, then parse that JSON data, and use the `session_id` provided by CoWIN to check if a notification for the current session was already sent or not, if not send a notification else skip it. 

In order to track changes in slots throughout the country(India) around 5.6k calls need to be make. CoWIN has about 800 districts on their website, and a call to their API only returns data for one district for one day, so in order to track for a week 5600 calls need to be made in a second. Note that this service can make upto 25k calls in a second(from our testing at least), but the biggest bottleneck is the CPU. So why is the CPU a bottleneck in this case? CoWIN APIs return a lot of data, for each call the data needs to be processed validated and then only can decisions be made, but we found that this processing and validating part was were we were having issues. So with a system with a higer core count and better memory management in the codebase, this process could be made a lot more effecient. 

Another issue that we faced was initially all the data was being stored in memory, which worked fine for a while, as we used to purge the slots that were over. But as slots slowly started increasing the entire process would crash due to low memory, so we decided to switch to using Redis. But there was another issue, making TCP calls locally was using up way too much resources again, so we switched redis from TCP to UNIX Sockets, which helped a lot, and got it to a stable state. But every few days the entire system would crash which we never ended up solving. 

#### Why did we try to make it so effeciant? 

We were essnetially trying to reduce our resouces as we were doing all this out of our pocket and some funding we got from DO, AWS, GCP, etc. 

End of the day we managed to run this entire service from duel core DO with 4GB RAM, for $5 a month. So we basically went above and beyond our requrirments as with the funding we got was more than enough to spin up 10 servers, but doing it with 1 and tight requirments is more fun as an engineer :)

###### Warning: This project if misued can cause damage. Use at your own discretion. We are not liable for misuse of this project.
