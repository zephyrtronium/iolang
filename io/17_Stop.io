Normal := Object clone do(
    isContinue := false
    isBreak := false
    stopLooping := false
    isReturn := false
    type := "Normal"
)

Eol := Object clone appendProto(Normal) do( // unused in iolang
    type := "Eol"
)

Continue := Object clone appendProto(Normal) do(
    isContinue := true
    type := "Continue"
)

Break := Object clone appendProto(Normal) do(
    isBreak := true
    stopLooping := true
    type := "Break"
)

Return := Object clone appendProto(Normal) do(
    isReturn := true
    stopLooping := true
)
