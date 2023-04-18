/**
 * - Soulteary Modify the JavaScript version for golang execution, under [Apache-2.0 license], 18/04/2023:
 *   - simplify the program, fix bugs, improve running speed, and allow running in golang
 *   - https://github.com/soulteary/nginx-formatter
 *
 * History:
 * - Yosef Ported the JavaScript beautifier under [Apache-2.0 license], 24/08/2016
 *   - https://github.com/vasilevich/nginxbeautifier
 * - Slomkowski Created a beautifier for nginx config files with Python under [Apache-2.0 license], 24/06/2016
 *   - https://github.com/1connect/nginx-config-formatter (https://github.com/slomkowski/nginx-config-formatter)
 */

/**
 * Grabs text in between two seperators seperator1 thetextIwant seperator2
 * @param {string} input String to seperate
 * @param {string} seperator1 The first seperator to use
 * @param {string} seperator2 The second seperator to use
 * @return {string}
 */
function extractTextBySeperator(input, seperator1, seperator2) {
  if (seperator2 == undefined) seperator2 = seperator1;
  var seperator1Regex = new RegExp(seperator1);
  var seperator2Regex = new RegExp(seperator2);
  var catchRegex = new RegExp(seperator1 + "(.*?)" + seperator2);
  if (seperator1Regex.test(input) && seperator2Regex.test(input)) {
    return input.match(catchRegex)[1];
  } else {
    return "";
  }
}

/**
 * Grabs text in between two seperators seperator1 thetextIwant seperator2
 * @param {string} input String to seperate
 * @param {string} seperator1 The first seperator to use
 * @param {string} seperator2 The second seperator to use
 * @return {object}
 */
function extractAllPossibleText(input, seperator1, seperator2) {
  if (seperator2 == undefined) seperator2 = seperator1;
  var extracted = {};
  var textInBetween;
  var cnt = 0;
  var seperator1CharCode = seperator1.length > 0 ? seperator1.charCodeAt(0) : "";
  var seperator2CharCode = seperator2.length > 0 ? seperator2.charCodeAt(0) : "";
  while ((textInBetween = extractTextBySeperator(input, seperator1, seperator2)) != "") {
    var placeHolder = "#$#%#$#placeholder" + cnt + "" + seperator1CharCode + "" + seperator2CharCode + "#$#%#$#";
    extracted[placeHolder] = seperator1 + textInBetween + seperator2;
    input = input.replace(extracted[placeHolder], placeHolder);
    cnt++;
  }
  return {
    filteredInput: input,
    extracted: extracted,
    getRestored: function () {
      var textToFix = this.filteredInput;
      for (var key in extracted) {
        textToFix = textToFix.replace(key, extracted[key]);
      }
      return textToFix;
    },
  };
}

/**
 * @param {string} single_line the whole nginx config
 * @return {string} stripped out string without multi spaces
 */
function strip_line(single_line) {
  //"""Strips the line and replaces neighbouring whitespaces with single space (except when within quotation marks)."""
  //trim the line before and after
  var trimmed = single_line.trim();
  //get text without any quatation marks(text foudn with quatation marks is replaced with a placeholder)
  var removedDoubleQuatations = extractAllPossibleText(trimmed, '"', '"');
  //replace multi spaces with single spaces, but skip in sub_filter directive
  if (!removedDoubleQuatations.filteredInput.includes("sub_filter")) {
    removedDoubleQuatations.filteredInput = removedDoubleQuatations.filteredInput.replace(/\s\s+/g, " ");
  }
  //restore anything of quatation marks
  return removedDoubleQuatations.getRestored();
}

/**
 * @param {string} configContents the whole nginx config
 */
function clean_lines(configContents) {
  var splittedByLines = configContents.split(/\r\n|\r|\n/g);
  //put {  } on their own seperate lines
  //trim the spaces before and after each line
  //trim multi spaces into single spaces
  //trim multi lines into two

  for (var index = 0, newline = 0; index < splittedByLines.length; index++) {
    splittedByLines[index] = splittedByLines[index].trim();
    if (splittedByLines[index] != "") {
      splittedByLines[index] = splittedByLines[index].replace(/\{\}/g, `{ }`);
    }

    if (!splittedByLines[index].startsWith("#") && splittedByLines[index] != "") {
      newline = 0;
      var line = (splittedByLines[index] = strip_line(splittedByLines[index]));
      if (line != "}" && line != "{" && !(line.includes("('{") || line.includes("}')") || line.includes("'{'") || line.includes("'}'"))) {
        var startOfComment = line.indexOf("#");
        var code = startOfComment >= 0 ? line.slice(0, startOfComment) : line;

        var removedDoubleQuatations = extractAllPossibleText(code, '"', '"');
        code = removedDoubleQuatations.filteredInput;

        var startOfParanthesis = code.indexOf("}");
        if (startOfParanthesis >= 0) {
          if (startOfParanthesis > 0) {
            splittedByLines[index] = strip_line(code.slice(0, startOfParanthesis - 1));
            splittedByLines.splice(index + 1, 0, "}");
          }
          var l2 = strip_line(code.slice(startOfParanthesis + 1));
          if (l2 != "") splittedByLines.splice(index + 2, 0, l2);
          code = splittedByLines[index];
        }
        var endOfParanthesis = code.indexOf("{");
        if (endOfParanthesis >= 0) {
          splittedByLines[index] = strip_line(code.slice(0, endOfParanthesis));
          splittedByLines.splice(index + 1, 0, "{");
          var l2 = strip_line(code.slice(endOfParanthesis + 1));
          if (l2 != "") splittedByLines.splice(index + 2, 0, l2);
        }

        removedDoubleQuatations.filteredInput = splittedByLines[index];
        line = removedDoubleQuatations.getRestored();
        splittedByLines[index] = line;
      }
    }
    //remove more than two newlines
    else if (splittedByLines[index] == "") {
      if (newline++ >= 2) {
        splittedByLines.splice(index, 1);
        index--;
      }
    }
  }
  return splittedByLines;
}

function join_opening_bracket(lines) {
  for (var i = 0; i < lines.length; i++) {
    var line = lines[i];
    if (line == "{") {
      //just make sure we don't put anything before 0
      if (i >= 1) {
        lines[i] = lines[i - 1] + " {";
        lines.splice(i - 1, 1);
      }
    }
  }
  return lines;
}

function fold_empty_brackets(lines) {
  return lines
    .join("\n")
    .replace(new RegExp(`\\s+{[\\s\\n\\r]*?}`, "gm"), " {  }")
    .replace(/\n{3,}/gm, "\n\n");
}

function add_empty_line_after_nginx_directives(lines) {
  let clone = lines.reverse();
  let output = [];
  for (let i = 0, j = clone.length; i < j; i++) {
    let current = (clone[i] + "").trim();
    let next = (clone[i + 1] + "").trim();
    if (next && !next.startsWith("}") && current.startsWith("}")) output.push("");
    output.push(clone[i]);
  }
  return output.reverse();
}

function fixDollarVar(lines) {
  const placeHolder = `[dollar]`;
  return lines.map((line) => {
    while (line.indexOf(placeHolder) !== -1) {
      line = line.replace(placeHolder, "$");
    }
    return line;
  });
}

var options = { INDENTATION: "\t" };

function perform_indentation(lines) {
  var indented_lines, current_indent, line;
  ("Indents the lines according to their nesting level determined by curly brackets.");
  indented_lines = [];
  current_indent = 0;
  var iterator1 = lines;
  for (var index1 = 0; index1 < iterator1.length; index1++) {
    line = iterator1[index1];
    if (!line.startsWith("#") && /.*?\}(\s*#.*)?$/.test(line) && current_indent > 0) {
      current_indent -= 1;
    }
    if (line !== "") {
      indented_lines.push(options.INDENTATION.repeat(current_indent) + line);
    } else {
      indented_lines.push("");
    }
    if (!line.startsWith("#") && /.*?\{(\s*#.*)?$/.test(line)) {
      current_indent += 1;
    }
  }
  return indented_lines;
}

function FormatNginxConf(text, indentation = "  ") {
  options["INDENTATION"] = indentation;

  let lines = clean_lines(text);
  lines = join_opening_bracket(lines);
  lines = perform_indentation(lines);
  lines = add_empty_line_after_nginx_directives(lines);
  lines = fixDollarVar(lines);
  return fold_empty_brackets(lines);
}
