module Timer exposing (Timer, cancel, empty, replace, schedule)

import Process
import Task


{-| Implements an identity to track an asynchronous timer.
-}
type Timer
    = Empty
    | Idle Int
    | Timer Int


empty : Timer
empty =
    Empty


schedule : (Timer -> msg) -> Timer -> Float -> Cmd msg
schedule message timer millis =
    Task.perform (always (message timer)) (Process.sleep millis)


{-| Replaces the provided timer with a newly created one.
-}
replace : Timer -> Timer
replace previous =
    case previous of
        Empty ->
            Timer 0

        Idle index ->
            Timer (next index)

        Timer index ->
            Timer (next index)


{-| Cancels the provided timer without creating a replacement.
-}
cancel : Timer -> Timer
cancel previous =
    case previous of
        Timer index ->
            Idle index

        _ ->
            previous


{-| Increments the timer identity, preventing integer overflow.
-}
next : Int -> Int
next index =
    if index > 2 ^ 30 then
        0

    else
        index + 1
