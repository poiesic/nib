-- pdf.lua: pandoc lua filter for PDF output via xelatex
-- Embedded in scrib binary and written to a temp file at build time.

-- Convert HTML <br> tags to proper line breaks in all output formats.
function RawInline(el)
  if el.format == "html" and el.text:match("^<br%s*/?>$") then
    return pandoc.LineBreak()
  end
end
