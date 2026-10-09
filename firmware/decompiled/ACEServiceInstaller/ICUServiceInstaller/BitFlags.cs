using System;

namespace ICUServiceInstaller;

[Flags]
internal enum BitFlags
{
	First = 1,
	Second = 2,
	Third = First | Second,
	Fourth = 4,
	Fifth = First | Fourth,
	Sixth = Second | Fourth,
	Seventh = Third | Fourth,
	Eighth = 8
}
