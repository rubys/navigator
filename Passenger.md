# Passenger notes:

* Debian comes out infrequently, and inevitably contains a backlevel of passenger. Example: https://packages.debian.org/sid/passenger
* Ruby aggressively updates their docker hub images to the latest debian.
* Passenger has releases out of sync with Debian: https://blog.phusion.nl/tag/passenger-releases/; these releases bin nginx versions

What this inevitably means is that there are periods of time when you can't run the latest release of Ruby and Passenger via docker hub images. Some manual worksrounds are possible (hard coding debian releae names in the dockerfile, but these changes are fragile. See:

https://github.com/rubys/showcase/commit/51650f6f4ce37981732bd6de27c5e7220f35ee83

