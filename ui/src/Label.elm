module Label exposing (view)

import Data.MessageHeader exposing (Label)
import Html exposing (Html, span, text)
import Html.Attributes exposing (class, style, title)


{-| Renders message labels as a row of inline tags, or nothing when the message has none.
-}
view : List Label -> Html msg
view labels =
    if List.isEmpty labels then
        text ""

    else
        span [ class "message-labels" ] (List.map tag labels)


{-| A single label.  The color is mixed with the theme's background for the tag fill and
border, and lightly into the theme's text color, so any color, including white or black,
stays readable in both light and dark mode.  Labels without a color use the theme default.
Each value is set as one style property, so anything that is not a single CSS color is
dropped by the browser.  The tooltip shows the description, or the full name when there is
no description, since long names are truncated.
-}
tag : Label -> Html msg
tag label =
    let
        mix amount base =
            "color-mix(in srgb, " ++ label.color ++ " " ++ amount ++ ", var(" ++ base ++ "))"

        colorAttrs =
            if label.color == "" then
                []

            else
                [ class "message-label-colored"
                , style "background-color" (mix "18%" "--bg-color")
                , style "border-color" (mix "45%" "--bg-color")
                , style "color" (mix "30%" "--primary-color")
                ]

        tooltip =
            if label.description == "" then
                label.name

            else
                label.description
    in
    span (class "message-label" :: title tooltip :: colorAttrs) [ text label.name ]
