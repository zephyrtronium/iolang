System do(
    errorNumber := 0

    maxRecycledObjects := 0
    recycledObjectCount := 0
    setMaxRecycledObjects := 0

    symbols := list

    getOptions := method(args,
        opts := Map clone
        unkeyed := list
        args foreach(i, arg,
            arg beginsWithSeq("--") ifFalse(unkeyed append(arg); continue)
            k := arg findSeq("=")
            if(k,
                name := arg exSlice(2, k)
                val := arg exSlice(k + 1)
            ,
                name := arg exSlice(2)
                val := ""
            )
            opts atPut(name, val)
        )
        unkeyed isEmpty ifFalse(opts atPut("", unkeyed))
        opts
    )

    userInterruptHandler := method(
        "\nreceived interrupt; exiting" println
        self exit
    )

    sleep := Object getSlot("wait")
)
