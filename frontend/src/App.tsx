import {useMemo, useState} from 'react';
import './App.css';
import {ImportThreadFile, RecalculateThreadCodes} from '../wailsjs/go/main/App';
import {threadcalc} from '../wailsjs/go/models';

type ThreadResult = {
    code: string;
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

const metersFormatter = new Intl.NumberFormat('ru-RU', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
});

function formatMeters(value: number) {
    return `${metersFormatter.format(value)} м`;
}

function App() {
    const [skeinLength, setSkeinLength] = useState(8);
    const [result, setResult] = useState<ImportResult | null>(null);
    const [error, setError] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const [codeDrafts, setCodeDrafts] = useState<Record<string, string>>({});
    const items = result?.items ?? [];
    const warnings = result?.warnings ?? [];

    const unknownColorCount = useMemo(() => {
        return items.filter((item) => !item.paletteFound).length;
    }, [items]);

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

    return (
        <main className="app-shell">
            <section className="toolbar" aria-label="Импорт">
                <div>
                    <h1>Подсчет нитей DMC</h1>
                    <div className="file-name">{result?.fileName ?? 'Файл не выбран'}</div>
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

            <section className="summary" aria-label="Сводка">
                <div className="metric">
                    <span>Цветов</span>
                    <strong>{items.length}</strong>
                </div>
                <div className="metric">
                    <span>Метраж</span>
                    <strong>{formatMeters(result?.totalMeters ?? 0)}</strong>
                </div>
                <div className="metric">
                    <span>Мотков</span>
                    <strong>{result?.totalSkeins ?? 0}</strong>
                </div>
                <div className="metric">
                    <span>Строк импорта</span>
                    <strong>{result?.rowsImported ?? 0}</strong>
                </div>
                <div className="metric">
                    <span>Без палитры</span>
                    <strong>{unknownColorCount}</strong>
                </div>
            </section>

            {result && (
                <section className="meta-strip" aria-label="Детали импорта">
                    <span>{result.encoding}</span>
                    <span>{result.beadRowsIgnored} строк бисера проигнорировано</span>
                    <span>{formatMeters(result.skeinLengthMeters)} в мотке</span>
                </section>
            )}

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
                                <td className="number-column">{formatMeters(item.meters)}</td>
                                <td className="number-column">{item.skeins}</td>
                                <td>
                                    <div className="notes">
                                        {notes.length ? notes.map((note) => (
                                            <span className="note" key={note}>{note}</span>
                                        )) : <span className="note note-muted">OK</span>}
                                    </div>
                                </td>
                            </tr>
                        )
                    }) : (
                        <tr>
                            <td colSpan={5} className="empty-state">Нет данных</td>
                        </tr>
                    )}
                    </tbody>
                </table>
            </section>
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

export default App;
