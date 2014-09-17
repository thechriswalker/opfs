The initial idea for this tools was when my wife didn't like how I organised
our digital photos, by date. She preferred by "event", but that's hard to do
programmatically. She also likes the filesystem (e.g. a "shared folder") with
all the photos, whereas I like a web interface. Desktop tools are great, but
I have multiple machines and I want the tagging/albums/metadata to move with
the photos as I backup.

## Lots of issues.

## Partial solution: webapp!

So I looked. Many gallery software, but none desgined with a large personal
photo collection in mind. None that allowed a simple filesystem copy for backup.
None that exported a filesystem I could share over SMB for my wife. None that
allowed me to self host my photo files. (photographer.io deserves a mention, but
it ONLY support S3 as a filestore -- I already pay for my server, why not use it's
storage? I backup to multiple locations...)

## Enter Camlistore.

So Camlistore does all this and more, but it's complex and tricky to configure,
and at the moment very much alpha software. It is awesome and I would love to
use it but I didn't understand it and documentation seemed lacking. Maybe I didn't
dig deep enough.

Anyway, Camlistore gave me a lot of ideas. The content-addressable storage for one.
That is a major win, as I cannot duplicate content now.

## Bring on OPFS

So OPFS is a tool to import all your photos/video from you camera/phone and have
a fully searchable web interface. You can back it up with a simple copy. You will
be able to mount a FUSE filesystem from it as well. But that is for later. My
original plan for exporting files was a tree of symlinks and I might just leave
it at that.

It has a web interface/api for tagging photos so you can create albums, soft-delete,
hard-delete, view slideshows, export a selection of items in a zip (e.g. for printing),
search for items, etc.

Not everything is implemented yet. but it's getting there.