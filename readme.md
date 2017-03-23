# log for golang package logg
## useage 
NewLogger().LoadConfig(filename)
## log config
<code>
logg.root.level = debug <br>
logg.root.callfile = true <br>
logg.appender.stdout = console<br>
logg.appender.stdout.level = debug<br>
<br>
logg.appender = "A1;A2"<br>
logg.appender.A1 = file<br>
logg.appender.A1.file = error.log<br>
logg.appender.A1.level = error<br>
logg.appender.A1.maxday = 0<br>
logg.appender.A1.maxsize = 0<br>
logg.appender.A1.daily = true<br>
logg.appender.A1.rotate = true<br>
<br>
logg.appender.A2 = file<br>
logg.appender.A2.file = debug.log<br>
logg.appender.A2.level = debug<br>

</code>
