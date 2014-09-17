PFS
====

Now that *could* stand for something hip like `Open Polymedia File System`, but it would not be true. OPFS is a system for storing photos and video clips (primarily) in such a way that they can be searched, retrieved and easily backed up alongside their meta-data. I wanted to do it to keep the photo/video I have taken of my children in a platform agnostic way. So it is named after them: Orson Patrick and Florcence Scarlett.

OPFS runs from a single binary, and needs a config file (defaulting to: `~/.opfs/config.json`) to tell it where to store data, and where to watch for new data.

It exposes an API used for the web client, and I hope at some point to be able to write a FUSE module for it so it can be accessed as a local filesystem.

