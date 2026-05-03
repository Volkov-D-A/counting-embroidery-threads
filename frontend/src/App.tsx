import {useMemo, useState} from 'react';
import './App.css';
import {ImportThreadFile, RecalculateThreadCodes} from '../wailsjs/go/main/App';
import {threadcalc} from '../wailsjs/go/models';

type ThreadResult = {
    code: string;
    colorName: string;
    colorHex: string;
    paletteFound: boolean;
    meters: number;
    skeins: number;
    notes: string[] | null;
};

type ImportResult = {
    cancelled: boolean;
    filePath: string;
    fileName: string;
    encoding: string;
    rowsImported: number;
    beadRowsIgnored: number;
    totalMeters: number;
    totalSkeins: number;
    skeinLengthMeters: number;
    items: ThreadResult[] | null;
    warnings: string[] | null;
};

type CodeCorrection = {
    from: string;
    to: string;
};

type SummaryLanguage = 'ru' | 'eng';

const metersFormatter = new Intl.NumberFormat('ru-RU', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
});

const compactMetersFormatter = new Intl.NumberFormat('ru-RU', {
    minimumFractionDigits: 0,
    maximumFractionDigits: 2,
});

function formatMeters(value: number) {
    return `${metersFormatter.format(value)} м`;
}

function formatCompactMeters(value: number) {
    return compactMetersFormatter.format(value);
}

function App() {
    const [skeinLength, setSkeinLength] = useState(8);
    const [result, setResult] = useState<ImportResult | null>(null);
    const [error, setError] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const [codeDrafts, setCodeDrafts] = useState<Record<string, string>>({});
    const [summaryLanguage, setSummaryLanguage] = useState<SummaryLanguage | null>(null);
    const [copyStatus, setCopyStatus] = useState('');
    const items = result?.items ?? [];
    const warnings = result?.warnings ?? [];

    const unknownColorCount = useMemo(() => {
        return items.filter((item) => !item.paletteFound).length;
    }, [items]);

    const summaryText = useMemo(() => {
        if (!summaryLanguage) {
            return '';
        }
        return formatSummary(items, summaryLanguage);
    }, [items, summaryLanguage]);

    async function importFile() {
        setIsLoading(true);
        setError('');
        try {
            const imported = await ImportThreadFile(skeinLength) as ImportResult;
            if (imported && !imported.cancelled) {
                setResult(normalizeResult(imported));
                setCodeDrafts({});
            }
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setIsLoading(false);
        }
    }

    async function recalculate(corrections: CodeCorrection[] = [], nextSkeinLength = skeinLength) {
        if (!result) {
            return;
        }

        setIsLoading(true);
        setError('');
        try {
            const recalculated = await RecalculateThreadCodes(
                result as unknown as threadcalc.ImportResult,
                corrections as threadcalc.CodeCorrection[],
                nextSkeinLength,
            ) as ImportResult;
            setResult(normalizeResult(recalculated));
            setCodeDrafts({});
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setIsLoading(false);
        }
    }

    async function commitCodeCorrection(item: ThreadResult) {
        const targetCode = (codeDrafts[item.code] ?? item.code).trim();
        if (!targetCode || targetCode === item.code) {
            setCodeDrafts((drafts) => {
                const next = {...drafts};
                delete next[item.code];
                return next;
            });
            return;
        }

        await recalculate([{from: item.code, to: targetCode}]);
    }

    function updateSkeinLength(value: number) {
        setSkeinLength(Number.isFinite(value) ? value : 8);
    }

    function openSummary(language: SummaryLanguage) {
        setCopyStatus('');
        setSummaryLanguage(language);
    }

    async function copySummary() {
        if (!summaryText) {
            return;
        }

        try {
            await copyToClipboard(summaryText);
            setCopyStatus('Скопировано');
        } catch (err) {
            setCopyStatus('Не удалось скопировать');
        }
    }

    return (
        <main className="app-shell">
            <section className="toolbar" aria-label="Импорт">
                <div className="toolbar-status">
                    <div className="file-name">{result?.fileName ?? 'Файл не выбран'}</div>
                    <section className="summary" aria-label="Сводка">
                        <div className="metric">
                            <span>Цветов</span>
                            <strong>{items.length}</strong>
                        </div>
                        <div className="metric">
                            <span>Строк</span>
                            <strong>{result?.rowsImported ?? 0}</strong>
                        </div>
                        <div className="metric">
                            <span>Без палитры</span>
                            <strong>{unknownColorCount}</strong>
                        </div>
                    </section>
                </div>

                <div className="toolbar-actions">
                    <label className="field">
                        <span>Длина мотка, м</span>
                        <input
                            type="number"
                            min="0.1"
                            step="0.1"
                            value={skeinLength}
                            onChange={(event) => updateSkeinLength(Number(event.target.value))}
                            onKeyDown={(event) => {
                                if (event.key === 'Enter') {
                                    recalculate([], skeinLength);
                                }
                            }}
                        />
                    </label>
                    <button className="secondary-button" onClick={() => recalculate()} disabled={isLoading || !result}>
                        Пересчитать
                    </button>
                    <button className="primary-button" onClick={importFile} disabled={isLoading}>
                        {isLoading ? 'Импорт...' : 'Открыть TXT'}
                    </button>
                </div>
            </section>

            {error && <div className="alert alert-error">{error}</div>}

            {warnings.length ? (
                <section className="warnings" aria-label="Предупреждения">
                    {warnings.map((warning) => (
                        <div className="alert" key={warning}>{warning}</div>
                    ))}
                </section>
            ) : null}

            <section className="table-wrap" aria-label="Расход нитей">
                <table>
                    <thead>
                    <tr>
                        <th className="swatch-column">Цвет</th>
                        <th>Код DMC</th>
                        <th>Название</th>
                        <th className="number-column">Метраж</th>
                        <th className="number-column">Мотки</th>
                        <th>Статус</th>
                    </tr>
                    </thead>
                    <tbody>
                    {items.length ? items.map((item) => {
                        const notes = item.notes ?? [];
                        return (
                            <tr key={item.code}>
                                <td className="swatch-column">
                                    <span
                                        className={`swatch ${item.paletteFound ? '' : 'swatch-missing'}`}
                                        style={{backgroundColor: item.colorHex}}
                                    />
                                </td>
                                <td className="code-cell">
                                    <input
                                        className="code-input"
                                        value={codeDrafts[item.code] ?? item.code}
                                        disabled={isLoading}
                                        aria-label={`Код DMC ${item.code}`}
                                        onChange={(event) => {
                                            const value = event.target.value;
                                            setCodeDrafts((drafts) => ({...drafts, [item.code]: value}));
                                        }}
                                        onBlur={() => commitCodeCorrection(item)}
                                        onKeyDown={(event) => {
                                            if (event.key === 'Enter') {
                                                event.currentTarget.blur();
                                            }
                                            if (event.key === 'Escape') {
                                                setCodeDrafts((drafts) => {
                                                    const next = {...drafts};
                                                    delete next[item.code];
                                                    return next;
                                                });
                                                event.currentTarget.blur();
                                            }
                                        }}
                                    />
                                </td>
                                <td className="name-cell">{item.colorName || 'не найдено'}</td>
                                <td className="number-column">{formatMeters(item.meters)}</td>
                                <td className="number-column">{item.skeins}</td>
                                <td>
                                    <div className="notes">
                                        {notes.length ? notes.map((note) => (
                                            <span className="note" key={note}>{note}</span>
                                        )) : <span className="note note-muted">ОК</span>}
                                    </div>
                                </td>
                            </tr>
                        )
                    }) : (
                        <tr>
                            <td colSpan={6} className="empty-state">Нет данных</td>
                        </tr>
                    )}
                    </tbody>
                </table>
            </section>

            <div className="floating-summary-actions" aria-label="Компактный результат">
                <button type="button" onClick={() => openSummary('ru')} disabled={!items.length}>
                    RU
                </button>
                <button type="button" onClick={() => openSummary('eng')} disabled={!items.length}>
                    ENG
                </button>
            </div>

            {summaryLanguage && (
                <div
                    className="modal-backdrop"
                    onMouseDown={(event) => {
                        if (event.target === event.currentTarget) {
                            setSummaryLanguage(null);
                        }
                    }}
                >
                    <section className="summary-modal" role="dialog" aria-modal="true" aria-label="Компактный результат">
                        <header className="modal-header">
                            <strong>{summaryLanguage === 'ru' ? 'RU' : 'ENG'}</strong>
                            <div className="modal-actions">
                                <span className="copy-status">{copyStatus}</span>
                                <button type="button" className="secondary-button" onClick={copySummary}>
                                    Копировать
                                </button>
                                <button type="button" className="icon-button" aria-label="Закрыть" onClick={() => setSummaryLanguage(null)}>
                                    ×
                                </button>
                            </div>
                        </header>
                        <pre className="summary-text">{summaryText}</pre>
                    </section>
                </div>
            )}
        </main>
    );
}

function normalizeResult(result: ImportResult): ImportResult {
    return {
        ...result,
        items: result.items ?? [],
        warnings: result.warnings ?? [],
    };
}

function formatSummary(items: ThreadResult[], language: SummaryLanguage) {
    const unit = language === 'ru' ? 'шт.' : 'pcs.';
    const meterUnit = language === 'ru' ? 'м' : 'm';

    return items
        .map((item) => `${item.code}   ${item.skeins}${unit} (${formatCompactMeters(item.meters)}${meterUnit})`)
        .join('\n');
}

async function copyToClipboard(text: string) {
    if (navigator.clipboard?.writeText) {
        try {
            await navigator.clipboard.writeText(text);
            return;
        } catch (err) {
            // Fall back to the legacy command below for stricter WebView clipboard settings.
        }
    }

    const textArea = document.createElement('textarea');
    textArea.value = text;
    textArea.setAttribute('readonly', 'true');
    textArea.style.position = 'fixed';
    textArea.style.left = '-9999px';
    document.body.appendChild(textArea);
    textArea.select();

    try {
        if (!document.execCommand('copy')) {
            throw new Error('copy command failed');
        }
    } finally {
        document.body.removeChild(textArea);
    }
}

export default App;
