package src

import killswitch "example.test/killswitch"

var invoiceMatcher = killswitch.Gate{Name: "invoice_matcher"}
var carrierPause = killswitch.Gate{DB: nil, Name: "invoice_matcher_carrier"}
