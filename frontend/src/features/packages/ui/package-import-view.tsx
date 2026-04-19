"use client"

import { useState, useCallback, useMemo, useEffect } from "react"
import Link from "next/link"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/ui/components/card"
import { Button } from "@/ui/components/button"
import { Badge } from "@/ui/components/badge"
import { Textarea } from "@/ui/components/textarea"
import { ArrowLeft, Upload, FileText, Loader2, AlertCircle, CheckCircle2, CloudUpload } from "lucide-react"
import { Checkbox } from "@/ui/components/checkbox"
import { Progress } from "@/ui/components/progress"
import { Alert, AlertDescription } from "@/ui/components/alert"
import { ScrollArea } from "@/ui/components/scroll-area"
import { useBulkImportPackages, usePackages } from "@/features/packages/hooks/use-packages"

type ImportFormat = "requirements_txt" | "package_json" | "list"

type ParsedEntry = {
  name: string
  ecosystem: string
  status: "new" | "exists" | "error"
  error?: string
  selected: boolean
}

const parseRequirementsTxt = (text: string): { name: string; ecosystem: string }[] => {
  return text
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => line && !line.startsWith("#") && !line.startsWith("-"))
    .map((line) => {
      const name = line.split(/[=<>~!\[;@]/)[0].trim()
      return { name, ecosystem: "python" }
    })
    .filter((p) => p.name.length > 0)
}

const parsePackageJson = (text: string): { name: string; ecosystem: string }[] => {
  try {
    const pkg = JSON.parse(text)
    const deps = { ...pkg.dependencies, ...pkg.devDependencies }
    return Object.keys(deps).map((name) => ({ name, ecosystem: "npm" }))
  } catch {
    return []
  }
}

const detectFormat = (text: string, fileName?: string): ImportFormat => {
  if (fileName?.endsWith(".json") || fileName === "package.json") {
    return "package_json"
  }
  if (text.trim().startsWith("{")) {
    return "package_json"
  }
  return "requirements_txt"
}

export const PackageImportView = () => {
  const [textInput, setTextInput] = useState("")
  const [detectedFormat, setDetectedFormat] = useState<ImportFormat>("requirements_txt")
  const [entries, setEntries] = useState<ParsedEntry[]>([])
  const [parseError, setParseError] = useState("")
  const [importResult, setImportResult] = useState<{
    imported: number
    skipped: number
    errors: { name: string; error: string }[]
  } | null>(null)
  const [dragActive, setDragActive] = useState(false)
  const [importProgress, setImportProgress] = useState<number | null>(null)

  const bulkImportMutation = useBulkImportPackages()

  // Fetch existing packages to cross-reference
  const { data: existingRes } = usePackages({ limit: 500 })
  const existingNames = useMemo(() => {
    const pkgs = existingRes?.data ?? []
    return new Set(pkgs.map((p) => `${p.name}:${p.ecosystem}`))
  }, [existingRes])

  // ─── Parsing helpers ───

  const buildEntries = useCallback(
    (raw: { name: string; ecosystem: string }[]): ParsedEntry[] => {
      return raw.map((pkg) => ({
        ...pkg,
        status: existingNames.has(`${pkg.name}:${pkg.ecosystem}`) ? "exists" as const : "new" as const,
        selected: !existingNames.has(`${pkg.name}:${pkg.ecosystem}`),
      }))
    },
    [existingNames]
  )

  const parseContent = useCallback(
    (text: string, fileName?: string) => {
      setParseError("")
      setImportResult(null)

      if (!text.trim()) {
        setParseError("No content to parse.")
        return
      }

      const format = detectFormat(text, fileName)
      setDetectedFormat(format)

      const packages =
        format === "package_json" ? parsePackageJson(text) : parseRequirementsTxt(text)

      if (packages.length === 0) {
        setParseError(
          format === "package_json"
            ? 'Could not parse any packages from this JSON. Ensure it has a "dependencies" or "devDependencies" field.'
            : "Could not parse any packages. Use requirements.txt format (one package per line) or a package.json file."
        )
        return
      }

      setEntries(buildEntries(packages))
    },
    [buildEntries]
  )

  // ─── File handling (click + drag-and-drop) ───

  const processFile = useCallback(
    (file: File) => {
      const reader = new FileReader()
      reader.onload = (event) => {
        const content = event.target?.result as string
        setTextInput(content)
        parseContent(content, file.name)
      }
      reader.readAsText(file)
    },
    [parseContent]
  )

  const handleFileUpload = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0]
      if (file) processFile(file)
    },
    [processFile]
  )

  const handleDragEnter = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setDragActive(true)
  }, [])

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    // Only deactivate if leaving the drop zone entirely
    if (e.currentTarget.contains(e.relatedTarget as Node)) return
    setDragActive(false)
  }, [])

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
  }, [])

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault()
      e.stopPropagation()
      setDragActive(false)

      const file = e.dataTransfer.files?.[0]
      if (file) processFile(file)
    },
    [processFile]
  )

  const handleTextParse = useCallback(() => {
    parseContent(textInput)
  }, [textInput, parseContent])

  const handlePaste = useCallback(
    (e: React.ClipboardEvent<HTMLTextAreaElement>) => {
      const pasted = e.clipboardData.getData("text")
      // After paste, the textarea value will be updated by React's onChange.
      // Use the pasted text directly for immediate parsing.
      const combined = textInput.slice(0, e.currentTarget.selectionStart) +
        pasted +
        textInput.slice(e.currentTarget.selectionEnd)
      // Defer so the textarea state updates first
      setTimeout(() => {
        setTextInput(combined)
        parseContent(combined)
      }, 0)
    },
    [textInput, parseContent]
  )

  // ─── Selection ───

  const selectedCount = entries.filter((e) => e.selected).length
  const newCount = entries.filter((e) => e.status === "new").length
  const existsCount = entries.filter((e) => e.status === "exists").length

  const allSelected = entries.length > 0 && entries.every((e) => e.selected)
  const someSelected = entries.some((e) => e.selected) && !allSelected

  const toggleSelectAll = () => {
    const nextVal = !allSelected
    setEntries((prev) => prev.map((e) => ({ ...e, selected: nextVal })))
  }

  const toggleEntry = (index: number) => {
    setEntries((prev) =>
      prev.map((e, i) => (i === index ? { ...e, selected: !e.selected } : e))
    )
  }

  // ─── Import ───

  const handleImport = async () => {
    const selected = entries.filter((e) => e.selected)
    if (selected.length === 0) return

    setImportProgress(0)

    // Simulate progress (actual import is a single POST)
    const progressInterval = setInterval(() => {
      setImportProgress((prev) => {
        if (prev === null || prev >= 90) return prev
        return prev + Math.random() * 15
      })
    }, 200)

    try {
      const result = await bulkImportMutation.mutateAsync({
        format: detectedFormat,
        content: textInput,
      })
      clearInterval(progressInterval)
      setImportProgress(100)
      setImportResult(result.data)
      setEntries([])
      setTextInput("")

      // Clear progress after a short delay
      setTimeout(() => setImportProgress(null), 1000)
    } catch {
      clearInterval(progressInterval)
      setImportProgress(null)
    }
  }

  // Re-compute entries when existingNames changes (lazy re-check)
  useEffect(() => {
    if (entries.length > 0) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- sync entry status with external data
      setEntries((prev) =>
        prev.map((e) => ({
          ...e,
          status: existingNames.has(`${e.name}:${e.ecosystem}`) ? "exists" : "new",
        }))
      )
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [existingNames])

  return (
    <div className="space-y-6">
      <div>
        <nav aria-label="Breadcrumb" className="mb-2">
          <ol className="flex items-center gap-1.5 text-sm text-muted-foreground">
            <li>
              <Link
                href="/packages"
                className="flex items-center gap-1 hover:text-foreground transition-colors"
              >
                <ArrowLeft className="h-3.5 w-3.5" />
                Packages
              </Link>
            </li>
            <li aria-hidden="true">/</li>
            <li className="text-foreground font-medium">Import</li>
          </ol>
        </nav>
        <h1 className="text-3xl font-bold">Bulk Import</h1>
        <p className="mt-1 text-muted-foreground">
          Import packages from requirements.txt or package.json files.
        </p>
      </div>

      {/* Import result banner */}
      {importResult && (
        <Card className="border-green-500/50">
          <CardContent className="py-4">
            <div className="flex items-start gap-3">
              <CheckCircle2 className="h-5 w-5 text-green-600 mt-0.5 shrink-0" />
              <div>
                <p className="font-medium text-green-600">Import completed</p>
                <p className="text-sm text-muted-foreground">
                  {importResult.imported} imported, {importResult.skipped} skipped
                  {importResult.errors.length > 0 && `, ${importResult.errors.length} errors`}
                </p>
                {importResult.errors.length > 0 && (
                  <ul className="mt-2 text-xs text-destructive space-y-1">
                    {importResult.errors.slice(0, 5).map((err, i) => (
                      <li key={i}>{err.name}: {err.error}</li>
                    ))}
                    {importResult.errors.length > 5 && (
                      <li>...and {importResult.errors.length - 5} more</li>
                    )}
                  </ul>
                )}
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* File Upload with drag-and-drop */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Upload className="h-5 w-5" />
              Upload File
            </CardTitle>
            <CardDescription>
              Upload a requirements.txt (Python) or package.json (NPM) file.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <label
              htmlFor="file-upload"
              className={`flex flex-col items-center justify-center rounded-lg border-2 border-dashed p-8 text-center cursor-pointer transition-colors ${
                dragActive
                  ? "border-blue-500 bg-blue-500/5"
                  : "border-muted-foreground/25 hover:border-muted-foreground/50"
              }`}
              onDragEnter={handleDragEnter}
              onDragLeave={handleDragLeave}
              onDragOver={handleDragOver}
              onDrop={handleDrop}
              role="button"
              tabIndex={0}
              aria-label="Upload file or drag and drop"
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault()
                  document.getElementById("file-upload")?.click()
                }
              }}
            >
              {dragActive ? (
                <>
                  <CloudUpload className="h-10 w-10 text-blue-500 mb-3 animate-bounce" />
                  <p className="text-sm font-medium text-blue-600">Drop to upload</p>
                </>
              ) : (
                <>
                  <FileText className="h-10 w-10 text-muted-foreground mb-3" />
                  <p className="text-sm font-medium">Click or drag to upload</p>
                  <p className="text-xs text-muted-foreground mt-1">
                    .txt, .json files accepted
                  </p>
                </>
              )}
              <input
                id="file-upload"
                type="file"
                accept=".txt,.json"
                className="sr-only"
                onChange={handleFileUpload}
              />
            </label>
          </CardContent>
        </Card>

        {/* Paste Input */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <FileText className="h-5 w-5" />
              Paste Content
            </CardTitle>
            <CardDescription>
              Paste package list content directly.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <Textarea
              className="min-h-[160px] font-mono"
              placeholder={`# requirements.txt format:\nrequests>=2.28.0\nflask==3.0.0\nnumpy\n\n# Or paste package.json content`}
              value={textInput}
              onChange={(e) => setTextInput(e.target.value)}
              onPaste={handlePaste}
              aria-label="Paste package list content"
            />
            <Button variant="outline" onClick={handleTextParse} className="w-full">
              Parse Packages
            </Button>
          </CardContent>
        </Card>
      </div>

      {/* Parse error */}
      {parseError && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{parseError}</AlertDescription>
        </Alert>
      )}

      {/* Progress bar during import */}
      {importProgress !== null && (
        <div className="space-y-1">
          <div className="flex items-center justify-between text-xs text-muted-foreground">
            <span>Importing packages...</span>
            <span>{Math.round(importProgress)}%</span>
          </div>
          <Progress
            value={Math.round(importProgress)}
            aria-label="Import progress"
          />
        </div>
      )}

      {/* Parsed preview with checkboxes */}
      {entries.length > 0 && (
        <Card>
          <CardHeader>
            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <CardTitle>
                  Parsed Packages ({entries.length} found)
                </CardTitle>
                <CardDescription className="mt-1">
                  {entries.length} package{entries.length !== 1 ? "s" : ""} found
                  {existsCount > 0 && `, ${existsCount} already monitored`}.
                  {" "}Detected format:{" "}
                  <Badge variant="outline" className="ml-1">
                    {detectedFormat === "package_json" ? "package.json" : "requirements.txt"}
                  </Badge>
                </CardDescription>
              </div>
              <Button
                onClick={handleImport}
                disabled={bulkImportMutation.isPending || selectedCount === 0}
              >
                {bulkImportMutation.isPending ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <Upload className="mr-2 h-4 w-4" />
                )}
                Add {selectedCount} Selected Package{selectedCount !== 1 ? "s" : ""}
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            {/* Select all header */}
            <div className="flex items-center gap-3 pb-3 mb-3 border-b">
              <Checkbox
                checked={allSelected}
                indeterminate={someSelected}
                onCheckedChange={toggleSelectAll}
                aria-label="Select all packages"
              />
              <span className="text-sm font-medium">
                Select All
              </span>
              <span className="text-xs text-muted-foreground ml-auto">
                {selectedCount} of {entries.length} selected
              </span>
            </div>

            {/* Package list */}
            <ScrollArea className="max-h-80" role="list" aria-label="Parsed packages">
              <div className="space-y-0.5">
              {entries.map((entry, i) => (
                <label
                  key={`${entry.name}-${entry.ecosystem}-${i}`}
                  className={`flex items-center gap-3 rounded-md px-2 py-1.5 cursor-pointer transition-colors hover:bg-muted/50 ${
                    entry.selected ? "bg-muted/30" : ""
                  }`}
                  role="listitem"
                >
                  <Checkbox
                    checked={entry.selected}
                    onCheckedChange={() => toggleEntry(i)}
                    aria-label={`Select ${entry.name}`}
                  />
                  <span className="text-sm font-mono truncate">{entry.name}</span>
                  <Badge variant="outline" className="shrink-0 text-xs">
                    {entry.ecosystem === "npm" ? "NPM" : entry.ecosystem === "python" ? "Python" : entry.ecosystem}
                  </Badge>
                  <StatusBadge status={entry.status} />
                </label>
              ))}
              </div>
            </ScrollArea>

            {/* Summary line */}
            <div className="flex flex-wrap items-center justify-between gap-2 pt-3 mt-3 border-t text-sm text-muted-foreground">
              <span>
                {newCount} new, {existsCount} already monitored
              </span>
              <div className="flex items-center gap-2">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    setEntries([])
                    setTextInput("")
                  }}
                >
                  Cancel
                </Button>
                <Button
                  size="sm"
                  onClick={handleImport}
                  disabled={bulkImportMutation.isPending || selectedCount === 0}
                >
                  {bulkImportMutation.isPending ? (
                    <Loader2 className="mr-2 h-3.5 w-3.5 animate-spin" />
                  ) : (
                    <Upload className="mr-2 h-3.5 w-3.5" />
                  )}
                  Add {selectedCount} Selected
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

// ─── Status Badge ───

const StatusBadge = ({ status }: { status: ParsedEntry["status"] }) => {
  switch (status) {
    case "new":
      return (
        <Badge variant="outline" className="ml-auto shrink-0 text-xs border-green-500/50 text-green-600 bg-green-500/10">
          new
        </Badge>
      )
    case "exists":
      return (
        <Badge variant="secondary" className="ml-auto shrink-0 text-xs">
          exists
        </Badge>
      )
    case "error":
      return (
        <Badge variant="destructive" className="ml-auto shrink-0 text-xs">
          error
        </Badge>
      )
  }
}
