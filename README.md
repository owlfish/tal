# tal - Go Implementation of Template Attribute Language

[TAL](http://docs.zope.org/zope2/zope2book/AppendixC.html) is a template language created by the Zope project and implemented in a number of [different languages](https://en.wikipedia.org/wiki/Template_Attribute_Language).

The tal library provides an implementation of TAL for Go.  Development is mostly completed, test coverage is 86% and most of the documentation is completed.  
Current status:

 * TAL - Fully implemented.
 * TALES - Fully implemented (there are no plans to implement python: and nocall:)
 * METAL - Fully implemented.

To Do list:

 * Add some METAL benchmarks.
 * Look at supporting METAL tags in addition to attributes.
 * Clean up the debug logging statements.
 * Add more examples.
 * Document the HTML5 input / output behaviours.

 

